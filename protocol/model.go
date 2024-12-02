package protocol

// Acknowledgements contains whether a nack ("NAK") was received, or a list of ACKs, and for which objects those apply.
// If Nack is true, Acks is always empty. If Nack is false, Acks may be non-empty.
// The objects returned in Acks are always requested. Not all requested objects are necessarily listed.
// Not all sent objects are included in the list, and it may even be empty even if a cut point is found. This is an optimisation by the Git server.
//
// Git documentation defines the format as:
//
//	acknowledgments = PKT-LINE("acknowledgments" LF)
//	    (nak | *ack)
//	    (ready)
//	ready = PKT-LINE("ready" LF)
//	nak = PKT-LINE("NAK" LF)
//	ack = PKT-LINE("ACK" SP obj-id LF)
type Acknowledgements struct {
	// Invariant: Nack == true => Acks == nil
	//            Nack == false => len(Acks) >= 0

	Nack bool
	// FIXME: Are obj-ids fine as strings? Do we want a more proper type for them?
	//    obj-id    =  40*(HEXDIGIT)
	Acks []string
}

// TODO: Do we want to parse the acknowledgements here?

type Shallowness string

const (
	Shallow   = Shallowness("shallow")
	Unshallow = Shallowness("unshallow")
)

// ShallowInfo is sent when a shallow fetch or clone is requested.
//
//	shallow-info section
//	* If the client has requested a shallow fetch/clone, a shallow
//	  client requests a fetch or the server is shallow then the
//	  server's response may include a shallow-info section.  The
//	  shallow-info section will be included if (due to one of the
//	  above conditions) the server needs to inform the client of any
//	  shallow boundaries or adjustments to the clients already
//	  existing shallow boundaries.
type ShallowInfo struct {
	Shallowness Shallowness
	// FIXME: obj-id type?
	Object string
}

// TODO: Parse ShallowInfo here?

type WantedRef struct {
	// FIXME: obj-id type?
	Object  string
	RefName RefName
}

// TODO: Parse WantedRef here?

type FetchResponse struct {
	// These fields are in order.

	Acks       Acknowledgements
	Shallow    []ShallowInfo
	WantedRefs []WantedRef
	// mariell: Intentionally excluding packfile-uris because I can't see us needing them.

	// The packfile contains the majority of the information we want.
	//
	//	packfile section
	//	* This section is only included if the client has sent 'want'
	//	  lines in its request and either requested that no more
	//	  negotiation be done by sending 'done' or if the server has
	//	  decided it has found a sufficient cut point to produce a
	//	  packfile.
	//
	//	Always begins with the section header "packfile".
	//
	//	The transmission of the packfile begins immediately after the section header.
	//
	//	The data transfer of the packfile is always multiplexed, using the same semantics of the side-band-64k capability from protocol version 1.
	//	This means that each packet, during the packfile data stream, is made up of a leading 4-byte pkt-line length (typical of the pkt-line format), followed by a 1-byte stream code, followed by the actual data.
	//
	//	The stream code can be one of:
	//	1 - pack data
	//	2 - progress messages
	//	3 - fatal error message just before stream aborts
	Packfile any // TODO
	// When encoded, a flush-pkt is presented here.
}
