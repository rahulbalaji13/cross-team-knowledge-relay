package state

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"github.com/go-redis/redis/v8"
)

type EscrowStatus string

const (
	StatusCreated          EscrowStatus = "CREATED"
	StatusHeld             EscrowStatus = "HELD"
	StatusMeetingConfirmed EscrowStatus = "MEETING_CONFIRMED"
	StatusReleased         EscrowStatus = "RELEASED"
	StatusRefunded         EscrowStatus = "REFUNDED"
)

type Escrow struct {
	ID        string       `gorm:"primaryKey"`
	BountyID  string       `gorm:"index"`
	ExpertID  string       `gorm:"index"`
	Status    EscrowStatus `gorm:"type:varchar(20)"`
	Amount    int          `gorm:"not null"`
	UpdatedAt time.Time
}

var (
	ErrDuplicateRequest       = errors.New("duplicate request: idempotency key exists")
	ErrInvalidStateTransition = errors.New("invalid state transition")
)

type EscrowService struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewEscrowService(db *gorm.DB, rdb *redis.Client) *EscrowService {
	return &EscrowService{db: db, redis: rdb}
}

// ReleaseFunds handles moving the escrow state from HELD/CONFIRMED to RELEASED safely.
func (s *EscrowService) ReleaseFunds(ctx context.Context, escrowID string, idempotencyKey string) error {
	// 1. Check Idempotency using Redis NX
	added, err := s.redis.SetNX(ctx, "idemp:"+idempotencyKey, "locked", 24*time.Hour).Result()
	if err != nil || !added {
		return ErrDuplicateRequest
	}

	// 2. Transactional DB Update
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	var escrow Escrow
	// SELECT FOR UPDATE to prevent race conditions
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&escrow, "id = ?", escrowID).Error; err != nil {
		return err
	}

	// 3. State Machine Transition Logic
	if escrow.Status != StatusMeetingConfirmed && escrow.Status != StatusHeld {
		return ErrInvalidStateTransition
	}

	escrow.Status = StatusReleased
	if err := tx.Save(&escrow).Error; err != nil {
		return err
	}

	// 4. Commit successful update
	return tx.Commit().Error
}
