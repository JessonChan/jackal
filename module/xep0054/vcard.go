/*
 * Copyright (c) 2018 Miguel Ángel Ortuño.
 * See the LICENSE file for more information.
 */

package xep0054

import (
	"github.com/ortuman/jackal/log"
	"github.com/ortuman/jackal/module/xep0030"
	"github.com/ortuman/jackal/router"
	"github.com/ortuman/jackal/storage"
	"github.com/ortuman/jackal/xmpp"
)

const mailboxSize = 2048

const vCardNamespace = "vcard-temp"

// VCard represents a vCard server stream module.
type VCard struct {
	actorCh    chan func()
	shutdownCh chan chan error
}

// New returns a vCard IQ handler module.
func New(disco *xep0030.DiscoInfo) *VCard {
	v := &VCard{
		actorCh:    make(chan func(), mailboxSize),
		shutdownCh: make(chan chan error),
	}
	go v.loop()
	if disco != nil {
		disco.RegisterServerFeature(vCardNamespace)
		disco.RegisterAccountFeature(vCardNamespace)
	}
	return v
}

// MatchesIQ returns whether or not an IQ should be
// processed by the vCard module.
func (x *VCard) MatchesIQ(iq *xmpp.IQ) bool {
	return (iq.IsGet() || iq.IsSet()) && iq.Elements().ChildNamespace("vCard", vCardNamespace) != nil
}

// ProcessIQ processes a vCard IQ taking according actions
// over the associated stream.
func (x *VCard) ProcessIQ(iq *xmpp.IQ, r *router.Router) {
	x.actorCh <- func() {
		x.processIQ(iq, r)
	}
}

// Shutdown shuts down vCard module.
func (x *VCard) Shutdown() error {
	c := make(chan error)
	x.shutdownCh <- c
	return <-c
}

// runs on it's own goroutine
func (x *VCard) loop() {
	for {
		select {
		case f := <-x.actorCh:
			f()
		case c := <-x.shutdownCh:
			c <- nil
			return
		}
	}
}

func (x *VCard) processIQ(iq *xmpp.IQ, r *router.Router) {
	vCard := iq.Elements().ChildNamespace("vCard", vCardNamespace)
	if vCard != nil {
		if iq.IsGet() {
			x.getVCard(vCard, iq, r)
			return
		} else if iq.IsSet() {
			x.setVCard(vCard, iq, r)
			return
		}
	}
	_ = r.Route(iq.BadRequestError())
}

func (x *VCard) getVCard(vCard xmpp.XElement, iq *xmpp.IQ, r *router.Router) {
	if vCard.Elements().Count() > 0 {
		_ = r.Route(iq.BadRequestError())
		return
	}
	toJID := iq.ToJID()
	resElem, err := storage.FetchVCard(toJID.Node())
	if err != nil {
		log.Errorf("%v", err)
		_ = r.Route(iq.InternalServerError())
		return
	}
	log.Infof("retrieving vcard... (%s/%s)", toJID.Node(), toJID.Resource())

	resultIQ := iq.ResultIQ()
	if resElem != nil {
		resultIQ.AppendElement(resElem)
	} else {
		// empty vCard
		resultIQ.AppendElement(xmpp.NewElementNamespace("vCard", vCardNamespace))
	}
	_ = r.Route(resultIQ)
}

func (x *VCard) setVCard(vCard xmpp.XElement, iq *xmpp.IQ, r *router.Router) {
	fromJID := iq.FromJID()
	toJID := iq.ToJID()
	if toJID.IsServer() || (toJID.Node() == fromJID.Node()) {
		log.Infof("saving vcard... (%s/%s)", toJID.Node(), toJID.Resource())

		err := storage.InsertOrUpdateVCard(vCard, toJID.Node())
		if err != nil {
			log.Error(err)
			_ = r.Route(iq.InternalServerError())
			return

		}
		_ = r.Route(iq.ResultIQ())
	} else {
		_ = r.Route(iq.ForbiddenError())
	}
}
