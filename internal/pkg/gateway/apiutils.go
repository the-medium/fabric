/*
Copyright 2021 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package gateway

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	gp "github.com/hyperledger/fabric-protos-go/gateway"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric/protoutil"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getChannelAndChaincodeFromSignedProposal(signedProposal *peer.SignedProposal) (string, string, bool, error) {
	if signedProposal == nil {
		return "", "", false, fmt.Errorf("a signed proposal is required")
	}
	proposal, err := protoutil.UnmarshalProposal(signedProposal.ProposalBytes)
	if err != nil {
		return "", "", false, err
	}
	header, err := protoutil.UnmarshalHeader(proposal.Header)
	if err != nil {
		return "", "", false, err
	}
	channelHeader, err := protoutil.UnmarshalChannelHeader(header.ChannelHeader)
	if err != nil {
		return "", "", false, err
	}
	payload, err := protoutil.UnmarshalChaincodeProposalPayload(proposal.Payload)
	if err != nil {
		return "", "", false, err
	}
	spec, err := protoutil.UnmarshalChaincodeInvocationSpec(payload.Input)
	if err != nil {
		return "", "", false, err
	}

	return channelHeader.ChannelId, spec.ChaincodeSpec.ChaincodeId.Name, len(payload.TransientMap) > 0, nil
}

func newRpcError(code codes.Code, message string, details ...proto.Message) error {
	st := status.New(code, message)
	if len(details) != 0 {
		std, err := st.WithDetails(details...)
		if err == nil {
			return std.Err()
		} // otherwise return the error without the details
	}
	return st.Err()
}

func wrappedRpcError(err error, message string, details ...proto.Message) error {
	statusErr := status.Convert(err)
	return newRpcError(statusErr.Code(), message+": "+statusErr.Message(), details...)
}

func toRpcError(err error, unknownCode codes.Code) error {
	errStatus, ok := status.FromError(err)
	if ok {
		return errStatus.Err()
	}

	errStatus = status.FromContextError(err)
	if errStatus.Code() != codes.Unknown {
		return errStatus.Err()
	}

	return status.Error(unknownCode, err.Error())
}

func errorDetail(e *endpointConfig, msg string) *gp.ErrorDetail {
	return &gp.ErrorDetail{Address: e.address, MspId: e.mspid, Message: msg}
}

func getResultFromProposalResponse(proposalResponse *peer.ProposalResponse) ([]byte, error) {
	responsePayload := &peer.ProposalResponsePayload{}
	if err := proto.Unmarshal(proposalResponse.GetPayload(), responsePayload); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize proposal response payload")
	}

	return getResultFromProposalResponsePayload(responsePayload)
}

func getResultFromProposalResponsePayload(responsePayload *peer.ProposalResponsePayload) ([]byte, error) {
	chaincodeAction := &peer.ChaincodeAction{}
	if err := proto.Unmarshal(responsePayload.GetExtension(), chaincodeAction); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize chaincode action")
	}

	return chaincodeAction.GetResponse().GetPayload(), nil
}
