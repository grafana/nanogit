package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/grafana/hackathon-2024-12-nanogit/client"
	"github.com/grafana/hackathon-2024-12-nanogit/protocol"
	"github.com/lmittmann/tint"
)

func main() {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelDebug})))

	if err := run(); err != nil {
		slog.Error("app run returned error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := client.New("https://github.com/grafana/git-ui-sync-demo")
	if err != nil {
		return err
	}

	{
		reply, err := c.SmartInfoRequest(ctx)
		if err != nil {
			return err
		}

		lines, _, err := protocol.ParsePacket(reply)
		if err != nil {
			return err
		}

		for _, line := range lines {
			slog.Info("response", "line", strings.TrimRight(string(line), "\n"))
		}

		// TODO(mem): parse the response and adjust the following requests accordingly.
	}

	pkt, err := protocol.FormatPackets(
		protocol.PacketLine("command=ls-refs\n"),
		protocol.PacketLine("object-format=sha1\n"))
	if err != nil {
		return err
	}

	refsData, err := c.SendCommands(ctx, pkt)
	if err != nil {
		return err
	}

	lines, remainder, err := protocol.ParsePacket(refsData)
	if err != nil {
		return err
	}

	for _, line := range lines {
		slog.Info("line in data", "line", string(line))
	}

	slog.Info("and here's the remainder", "remainder", remainder)

	pkt, err = protocol.FormatPackets(
		// https://git-scm.com/docs/protocol-v2#_fetch
		protocol.PacketLine("command=fetch\n"),
		protocol.PacketLine("object-format=sha1\n"), // https://git-scm.com/docs/protocol-v2#_object_format
		protocol.DelimeterPacket,
		// thin-pack
		// Request that a thin pack be sent, which is a pack with deltas
		// which reference base objects not contained within the pack (but
		// are known to exist at the receiving end). This can reduce the
		// network traffic significantly, but it requires the receiving end
		// to know how to "thicken" these packs by adding the missing bases
		// to the pack.
		// protocol.PacketLine("thin-pack\n"),
		// no-progress
		// Request that progress information that would normally be sent on
		// side-band channel 2, during the packfile transfer, should not be
		// sent.  However, the side-band channel 3 is still used for error
		// responses.
		// TODO: What is a side-band channel in git's protocol??
		//   Relevant on side-bands: https://git-scm.com/docs/gitprotocol-pack#_packfile_data
		protocol.PacketLine("no-progress\n"),
		// filter <filter-spec>
		// Request that various objects from the packfile be omitted
		// using one of several filtering techniques. These are intended
		// for use with partial clone and partial fetch operations. See
		// `rev-list` for possible "filter-spec" values. When communicating
		// with other processes, senders SHOULD translate scaled integers
		// (e.g. "1k") into a fully-expanded form (e.g. "1024") to aid
		// interoperability with older receivers that may not understand
		// newly-invented scaling suffixes. However, receivers SHOULD
		// accept the following suffixes: 'k', 'm', and 'g' for 1024,
		// 1048576, and 1073741824, respectively.
		protocol.PacketLine("filter blob:none\n"),
		// want <oid>
		// Indicates to the server an object which the client wants to
		// retrieve.  Wants can be anything and are not limited to
		// advertised objects.
		//
		// The value here is a commit: https://github.com/grafana/git-ui-sync-demo/commit/6c86a0cdfd220c2fe3518cfaa4a4babf030d9a7a
		protocol.PacketLine("want 6c86a0cdfd220c2fe3518cfaa4a4babf030d9a7a\n"),
		// done
		// Indicates to the server that negotiation should terminate (or
		// not even begin if performing a clone) and that the server should
		// use the information supplied in the request to construct the
		// packfile.
		protocol.PacketLine("done\n"),
	)
	if err != nil {
		return err
	}

	out, err := c.SendCommands(ctx, pkt)
	if err != nil {
		return err
	}

	// TODO(mem): do something with the remaing data.
	lines, _, err = protocol.ParsePacket(out)
	if err != nil {
		return err
	}

	// The format of the output here is:
	//
	//     output = acknowledgements flush-pkt |
	//         [acknowledgments delim-pkt] [shallow-info delim-pkt]
	//         [wanted-refs delim-pkt] [packfile-uris delim-pkt]
	//         packfile flush-pkt
	//
	//     acknowledgments = PKT-LINE("acknowledgments" LF)
	//        (nak | *ack)
	//        (ready)
	//     ready = PKT-LINE("ready" LF)
	//     nak = PKT-LINE("NAK" LF)
	//     ack = PKT-LINE("ACK" SP obj-id LF)
	//
	//     shallow-info = PKT-LINE("shallow-info" LF)
	//        *PKT-LINE((shallow | unshallow) LF)
	//     shallow = "shallow" SP obj-id
	//     unshallow = "unshallow" SP obj-id
	//
	//     wanted-refs = PKT-LINE("wanted-refs" LF)
	//         *PKT-LINE(wanted-ref LF)
	//     wanted-ref = obj-id SP refname
	//
	//     packfile-uris = PKT-LINE("packfile-uris" LF) *packfile-uri
	//     packfile-uri = PKT-LINE(40*(HEXDIGIT) SP *%x20-ff LF)
	//
	//     packfile = PKT-LINE("packfile" LF)
	//         *PKT-LINE(%x01-03 *%x00-ff)
	response, err := protocol.ParseFetchResponse(lines)
	if err != nil {
		return err
	}

	slog.Info("fetch response", "parsed", response)

	for {
		obj, err := response.Packfile.ReadObject()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if obj.Object != nil {
			if obj.Object.Commit != nil {
				slog.Info("commit was read", "commit", *obj.Object.Commit)
			} else if obj.Object.Tree != nil {
				slog.Info("tree was read", "tree", obj.Object.Tree)
			} else if obj.Object.Delta != nil {
				slog.Info("delta was read", "delta", *obj.Object.Delta)
			} else {
				slog.Info("object was read",
					"ty", obj.Object.Type,
					"data", obj.Object.Data)
			}
		} else {
			slog.Info("trailer was read")
			break
		}
	}

	return nil
}
