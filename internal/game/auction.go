package game

import (
	"errors"
)

// AuctionState represents the state of a power plant auction
type AuctionState struct {
	CurrentPlant    *PowerPlant
	CurrentBidder   string
	CurrentBid      int
	ParticipantIDs  []string
	RoundComplete   bool
	PlantsPurchased int
}

// NewAuctionState creates a new auction state
func NewAuctionState() *AuctionState {
	return &AuctionState{
		CurrentPlant:    nil,
		CurrentBidder:   "",
		CurrentBid:      0,
		ParticipantIDs:  []string{},
		RoundComplete:   false,
		PlantsPurchased: 0,
	}
}

// StartAuction starts an auction for a power plant
func (g *Game) StartAuction(plantID int) error {
	// Find the plant in the market
	var plant *PowerPlant
	for _, p := range g.PowerPlants {
		if p.ID == plantID && p.InMarket {
			plant = p
			break
		}
	}

	if plant == nil {
		return errors.New("plant not found in market")
	}

	// Initialize auction state
	g.AuctionState = NewAuctionState()
	g.AuctionState.CurrentPlant = plant
	g.AuctionState.CurrentBid = plant.Cost // Start bid is the plant's cost

	// Reset all players for the auction
	for _, player := range g.Players {
		player.Reset()
		player.IsActive = true
	}

	// Copy active player IDs to participant list
	for _, id := range g.TurnOrder {
		g.AuctionState.ParticipantIDs = append(g.AuctionState.ParticipantIDs, id)
	}

	// Set the first player as the initial bidder
	if len(g.AuctionState.ParticipantIDs) > 0 {
		g.AuctionState.CurrentBidder = g.AuctionState.ParticipantIDs[0]
	}

	return nil
}

// ProcessBid processes a bid from a player
func (g *Game) ProcessBid(playerID string, plantID int, bid int) error {
	// Verify we're in an auction
	if g.AuctionState == nil || g.AuctionState.CurrentPlant == nil {
		return errors.New("no active auction")
	}

	// Verify this plant is being auctioned
	if g.AuctionState.CurrentPlant.ID != plantID {
		return errors.New("this plant is not being auctioned")
	}

	// Verify it's the player's turn to bid
	if g.AuctionState.CurrentBidder != playerID {
		return errors.New("not your turn to bid")
	}

	// Check if player is passing
	if bid == 0 {
		return g.PassOnBid(playerID)
	}

	// Verify the bid is higher than the current bid
	if bid <= g.AuctionState.CurrentBid {
		return errors.New("bid must be higher than current bid")
	}

	// Verify the player has enough money
	player := g.Players[playerID]
	if !player.HasEnoughMoney(bid) {
		return errors.New("not enough money for this bid")
	}

	// Update the auction state
	g.AuctionState.CurrentBid = bid
	player.CurrentBid = bid

	// Move to the next bidder
	return g.NextBidder()
}

// PassOnBid handles a player passing on a bid
func (g *Game) PassOnBid(playerID string) error {
	// Mark the player as having passed
	player := g.Players[playerID]
	player.HasPassed = true
	player.IsActive = false

	// Remove player from participants
	for i, id := range g.AuctionState.ParticipantIDs {
		if id == playerID {
			g.AuctionState.ParticipantIDs = append(
				g.AuctionState.ParticipantIDs[:i],
				g.AuctionState.ParticipantIDs[i+1:]...,
			)
			break
		}
	}

	// If only one player remains, they win the auction
	if len(g.AuctionState.ParticipantIDs) == 1 {
		winnerID := g.AuctionState.ParticipantIDs[0]
		return g.EndAuction(winnerID)
	}

	// Otherwise, move to the next bidder
	return g.NextBidder()
}

// NextBidder moves to the next bidder in the auction
func (g *Game) NextBidder() error {
	// Find the current bidder's position
	currentIndex := -1
	for i, id := range g.AuctionState.ParticipantIDs {
		if id == g.AuctionState.CurrentBidder {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return errors.New("current bidder not found in participants")
	}

	// Move to the next bidder
	nextIndex := (currentIndex + 1) % len(g.AuctionState.ParticipantIDs)
	g.AuctionState.CurrentBidder = g.AuctionState.ParticipantIDs[nextIndex]

	return nil
}

// EndAuction ends the current auction
func (g *Game) EndAuction(winnerID string) error {
	// Give the plant to the winner
	winner := g.Players[winnerID]
	if !winner.SpendMoney(g.AuctionState.CurrentBid) {
		return errors.New("winner doesn't have enough money")
	}

	// Remove the plant from the market and give it to the player
	plant := g.AuctionState.CurrentPlant
	winner.AddPowerPlant(plant)

	// Update auction state
	g.AuctionState.PlantsPurchased++
	g.AuctionState.CurrentPlant = nil
	g.AuctionState.CurrentBid = 0

	// Check if all players have purchased a plant or passed
	allDone := true
	for _, player := range g.Players {
		if player.IsActive && !player.HasPassed {
			allDone = false
			break
		}
	}

	if allDone {
		g.AuctionState.RoundComplete = true
	}

	return nil
}

// InitializeAuctionMarket initializes the auction market
func (g *Game) InitializeAuctionMarket() {
	// Sort power plants by cost
	sortPowerPlants(g.PowerPlants)

	// Mark first 8 plants as in market
	marketCount := 0
	for _, plant := range g.PowerPlants {
		if marketCount < 8 {
			plant.InMarket = true
			marketCount++
		} else {
			plant.InMarket = false
		}
	}
}
