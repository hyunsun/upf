// SPDX-License-Identifier: Apache-2.0
// Copyright 2022 Open Networking Foundation

package integration

import (
	"net"
	"testing"
	"time"

	"github.com/omec-project/pfcpsim/pkg/pfcpsim/session"
	"github.com/omec-project/upf-epc/test/integration/providers"
	"github.com/stretchr/testify/require"
	"github.com/wmnsk/go-pfcp/ie"
)

func TestUPFBasedUeIPAllocation(t *testing.T) {
	setup(t, ConfigUPFBasedIPAllocation)
	defer teardown(t)

	tc := testCase{
		ctx: testContext{
			UPFBasedUeIPAllocation: true,
		},
		input: &pfcpSessionData{
			sliceID:      1,
			nbAddress:    nodeBAddress,
			ueAddress:    ueAddress,
			upfN3Address: upfN3Address,
			sdfFilter:    "permit out ip from any to assigned",
			ulTEID:       15,
			dlTEID:       16,
			QFI:          0x09,
		},
		expected: p4RtValues{
			// first IP address from pool configured in ue_ip_alloc.json
			ueAddress: "10.250.0.1",
			// no application filtering rule expected
			appID:        0,
			tunnelPeerID: 2,
		},
		desc: "UPF-based UE IP allocation",
	}

	t.Run(tc.desc, func(t *testing.T) {
		testUEAttachDetach(t, fillExpected(&tc))
	})
}

func TestBasicPFCPAssociation(t *testing.T) {
	setup(t, ConfigDefault)
	defer teardown(t)

	err := pfcpClient.SetupAssociation()
	require.NoErrorf(t, err, "failed to setup PFCP association")

	time.Sleep(time.Second * 10)

	require.True(t, pfcpClient.IsAssociationAlive())
}

func TestSingleUEAttachAndDetach(t *testing.T) {
	setup(t, ConfigDefault)
	defer teardown(t)

	// Application filtering test cases
	testCases := []testCase{
		{
			input: &pfcpSessionData{
				sliceID:      1,
				nbAddress:    nodeBAddress,
				ueAddress:    ueAddress,
				upfN3Address: upfN3Address,
				sdfFilter:    "permit out udp from any 80-80 to assigned",
				ulTEID:       15,
				dlTEID:       16,
				QFI:          0x9,
			},
			expected: p4RtValues{
				appFilter: appFilter{
					proto:        0x11,
					appIP:        net.ParseIP("0.0.0.0"),
					appPrefixLen: 0,
					appPort: portRange{
						80, 80,
					},
				},
				appID:        1,
				tunnelPeerID: 2,
			},
			desc: "APPLICATION FILTERING permit out udp from any 80-80 to assigned",
		},
		{
			input: &pfcpSessionData{
				sliceID:      1,
				nbAddress:    nodeBAddress,
				ueAddress:    ueAddress,
				upfN3Address: upfN3Address,
				sdfFilter:    "permit out udp from 192.168.1.1/32 to assigned 80-400",
				ulTEID:       15,
				dlTEID:       16,
				QFI:          0x9,
			},
			expected: p4RtValues{
				appFilter: appFilter{
					proto:        0x11,
					appIP:        net.ParseIP("192.168.1.1"),
					appPrefixLen: 32,
					appPort: portRange{
						80, 400,
					},
				},
				// FIXME: there is a dependency on previous test because pfcpiface doesn't clear application IDs properly
				//  See SDFAB-960
				appID:        2,
				tunnelPeerID: 2,
			},
			desc: "APPLICATION FILTERING permit out udp from 192.168.1.1/32 to assigned 80-80",
		},
		{
			input: &pfcpSessionData{
				sliceID:      1,
				nbAddress:    nodeBAddress,
				ueAddress:    ueAddress,
				upfN3Address: upfN3Address,
				sdfFilter:    "permit out ip from any to assigned",
				ulTEID:       15,
				dlTEID:       16,
				QFI:          0x9,
			},
			expected: p4RtValues{
				// no application filtering rule expected
				appID:        0,
				tunnelPeerID: 2,
			},
			desc: "APPLICATION FILTERING ALLOW_ALL",
		},
		{
			input: &pfcpSessionData{
				sliceID:      1,
				nbAddress:    nodeBAddress,
				ueAddress:    ueAddress,
				upfN3Address: upfN3Address,
				sdfFilter:    defaultSDFFilter,
				ulTEID:       15,
				dlTEID:       16,

				QFI:              0x11,
				uplinkAppQerID:   1,
				downlinkAppQerID: 2,
				sessQerID:        4,
				sessGBR:          0,
				sessMBR:          500000,
				appGBR:           30000,
				appMBR:           50000,
			},
			expected: p4RtValues{
				appFilter: appFilter{
					proto:        0x11,
					appIP:        net.ParseIP("0.0.0.0"),
					appPrefixLen: 0,
					appPort: portRange{
						80, 80,
					},
				},
				appID:        1,
				tunnelPeerID: 2,
			},
			desc: "QER_METERING - 1 session QER, 2 app QERs",
		},
		{
			input: &pfcpSessionData{
				sliceID:      1,
				nbAddress:    nodeBAddress,
				ueAddress:    ueAddress,
				upfN3Address: upfN3Address,
				sdfFilter:    defaultSDFFilter,
				ulTEID:       15,
				dlTEID:       16,
				QFI:          0x11,

				// indicates 5G case (no application QERs, only session QER)
				uplinkAppQerID:   0,
				downlinkAppQerID: 0,

				sessQerID: 4,
				sessGBR:   300000,
				sessMBR:   500000,
			},
			expected: p4RtValues{
				appFilter: appFilter{
					proto:        0x11,
					appIP:        net.ParseIP("0.0.0.0"),
					appPrefixLen: 0,
					appPort: portRange{
						80, 80,
					},
				},
				appID:        1,
				tunnelPeerID: 2,
			},
			desc: "QER_METERING - session QER only",
		},
		{
			input: &pfcpSessionData{
				sliceID:      1,
				nbAddress:    nodeBAddress,
				ueAddress:    ueAddress,
				upfN3Address: upfN3Address,
				sdfFilter:    defaultSDFFilter,
				ulTEID:       15,
				dlTEID:       16,

				QFI:              0x08,
				uplinkAppQerID:   1,
				downlinkAppQerID: 2,
				sessQerID:        4,
				sessGBR:          0,
				sessMBR:          500000,
				appGBR:           30000,
				appMBR:           50000,
			},
			expected: p4RtValues{
				appFilter: appFilter{
					proto:        0x11,
					appIP:        net.ParseIP("0.0.0.0"),
					appPrefixLen: 0,
					appPort: portRange{
						80, 80,
					},
				},
				appID:        1,
				tunnelPeerID: 2,
				tc:           3,
			},
			desc: "QER_METERING - TC for QFI",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			testUEAttachDetach(t, fillExpected(&tc))
		})
	}
}

func fillExpected(tc *testCase) *testCase {
	if tc.expected.ueAddress == "" {
		tc.expected.ueAddress = tc.input.ueAddress
	}

	return tc
}

func testUEAttachDetach(t *testing.T, testcase *testCase) {
	err := pfcpClient.SetupAssociation()
	require.NoErrorf(t, err, "failed to setup PFCP association")

	pdrs := []*ie.IE{
		session.NewPDRBuilder().MarkAsUplink().
			WithMethod(session.Create).
			WithID(1).
			WithTEID(testcase.input.ulTEID).
			WithN3Address(testcase.input.upfN3Address).
			WithSDFFilter(testcase.input.sdfFilter).
			WithFARID(1).
			AddQERID(4).
			AddQERID(1).BuildPDR(),
	}

	if !testcase.ctx.UPFBasedUeIPAllocation {
		pdrs = append(pdrs,
			session.NewPDRBuilder().MarkAsDownlink().
				WithMethod(session.Create).
				WithID(2).
				WithUEAddress(testcase.input.ueAddress).
				WithSDFFilter(testcase.input.sdfFilter).
				WithFARID(2).
				AddQERID(4).
				AddQERID(2).BuildPDR())
	} else {
		pdrs = append(pdrs,
			// TODO: should be replaced by builder?
			ie.NewCreatePDR(
				ie.NewPDRID(2),
				ie.NewPrecedence(testcase.input.precedence),
				ie.NewPDI(
					ie.NewSourceInterface(ie.SrcInterfaceCore),
					// indicate UP to allocate UE IP Address
					ie.NewUEIPAddress(0x10, "", "", 0, 0),
					ie.NewSDFFilter(testcase.input.sdfFilter, "", "", "", 1),
				),
				ie.NewFARID(2),
				ie.NewQERID(2),
				ie.NewQERID(4),
			),
		)
	}

	fars := []*ie.IE{
		session.NewFARBuilder().
			WithMethod(session.Create).WithID(1).WithDstInterface(ie.DstInterfaceCore).
			WithAction(ActionForward).BuildFAR(),
		session.NewFARBuilder().
			WithMethod(session.Create).WithID(2).
			WithDstInterface(ie.DstInterfaceAccess).
			WithAction(ActionDrop).WithTEID(testcase.input.dlTEID).
			WithDownlinkIP(testcase.input.nbAddress).BuildFAR(),
	}

	var qers []*ie.IE
	if testcase.input.sessQerID != 0 {
		qers = append(qers,
			// session QER
			session.NewQERBuilder().WithMethod(session.Create).WithID(testcase.input.sessQerID).
				WithQFI(testcase.input.QFI).
				WithUplinkMBR(500000).
				WithDownlinkMBR(500000).
				WithUplinkGBR(0).
				WithDownlinkGBR(0).Build())
	}

	if testcase.input.uplinkAppQerID != 0 {
		qers = append(qers,
			// uplink application QER
			session.NewQERBuilder().WithMethod(session.Create).WithID(testcase.input.uplinkAppQerID).
				WithQFI(testcase.input.QFI).
				WithUplinkMBR(50000).
				WithDownlinkMBR(50000).
				WithUplinkGBR(30000).
				WithDownlinkGBR(30000).Build())
	}

	if testcase.input.downlinkAppQerID != 0 {
		qers = append(qers,
			// downlink application QER
			session.NewQERBuilder().WithMethod(session.Create).WithID(testcase.input.downlinkAppQerID).
				WithQFI(testcase.input.QFI).
				WithUplinkMBR(50000).
				WithDownlinkMBR(50000).
				WithUplinkGBR(30000).
				WithDownlinkGBR(30000).Build())
	}

	sess, err := pfcpClient.EstablishSession(pdrs, fars, qers)
	require.NoErrorf(t, err, "failed to establish PFCP session")

	verifyEntries(t, testcase.input, testcase.expected, false)

	err = pfcpClient.ModifySession(sess, nil, []*ie.IE{
		session.NewFARBuilder().
			WithMethod(session.Update).WithID(2).
			WithAction(ActionForward).WithDstInterface(ie.DstInterfaceAccess).
			WithTEID(testcase.input.dlTEID).WithDownlinkIP(testcase.input.nbAddress).BuildFAR(),
	}, nil)

	verifyEntries(t, testcase.input, testcase.expected, true)

	err = pfcpClient.DeleteSession(sess)
	require.NoErrorf(t, err, "failed to delete PFCP session")

	err = pfcpClient.TeardownAssociation()
	require.NoErrorf(t, err, "failed to gracefully release PFCP association")

	verifyNoEntries(t, testcase.expected)

	if isFastpathUP4() {
		// clear Applications table
		// FIXME: Temporary solution. They should be cleared by pfcpiface, see SDFAB-960
		p4rtClient, _ := providers.ConnectP4rt("127.0.0.1:50001", TimeBasedElectionId())
		defer func() {
			providers.DisconnectP4rt()
			// give pfcpiface time to become master controller again
			time.Sleep(3 * time.Second)
		}()
		entries, _ := p4rtClient.ReadTableEntryWildcard("PreQosPipe.applications")
		for _, entry := range entries {
			p4rtClient.DeleteTableEntry(entry)
		}
	}
}
