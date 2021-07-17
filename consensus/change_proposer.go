package consensus

import (
	"github.com/zarbchain/zarb-go/consensus/proposal"
	"github.com/zarbchain/zarb-go/consensus/vote"
	"github.com/zarbchain/zarb-go/crypto"
)

type changeProposerState struct {
	*consensus
}

func (s *changeProposerState) enter() {
	s.logger.Info("Requesting for changing proposer", "proposer", s.proposer(s.round).Address())
	s.signAddVote(vote.VoteTypeChangeProposer, crypto.UndefHash)

	s.decide()
}

func (s *changeProposerState) decide() {
	voteset := s.log.ChangeProposerVoteSet(s.round)
	if voteset.QuorumHash() != nil {
		s.logger.Debug("change proposer has quorum", "proposer", s.proposer(s.round).Address())
		s.round++

		s.enterNewState(s.proposeState)
	}
}

func (s *changeProposerState) onAddVote(v *vote.Vote) {
	// Only accept change propser votes
	if v.Type() == vote.VoteTypeChangeProposer {
		s.doAddVote(v)
		s.decide()
	}
}

func (s *changeProposerState) onSetProposal(p *proposal.Proposal) {
	// Ignore proposals
}

func (s *changeProposerState) onTimedout(t *ticker) {
	// Ignore timeouts
}

func (s *changeProposerState) name() string {
	return "change-proposer"
}
