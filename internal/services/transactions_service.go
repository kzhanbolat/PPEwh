package services

import (
	"errors"
	"fmt"
	"sort"

	"ppewh/internal/models"
	"ppewh/internal/storage"
)

type TransactionsService struct {
	itemsRepo        storage.ItemsRepository
	usersRepo        storage.UsersRepository
	transactionsRepo storage.TransactionsRepository
	returnsRepo      storage.ReturnsRepository
}

func NewTransactionsService(
	itemsRepo storage.ItemsRepository,
	usersRepo storage.UsersRepository,
	transactionsRepo storage.TransactionsRepository,
	returnsRepo storage.ReturnsRepository,
) *TransactionsService {
	return &TransactionsService{
		itemsRepo:        itemsRepo,
		usersRepo:        usersRepo,
		transactionsRepo: transactionsRepo,
		returnsRepo:      returnsRepo,
	}
}

func (s *TransactionsService) List() ([]models.Transaction, error) {
	txs, err := s.transactionsRepo.List()
	if err != nil {
		return nil, err
	}
	// Latest first; ISO-8601 timestamp sorts lexicographically.
	sort.Slice(txs, func(i, j int) bool { return txs[i].Timestamp > txs[j].Timestamp })
	return txs, nil
}

func (s *TransactionsService) GetByID(transactionID string) (models.Transaction, bool, error) {
	return s.transactionsRepo.GetByID(transactionID)
}

func (s *TransactionsService) ListReturns() ([]models.Return, error) {
	return s.returnsRepo.List()
}

// IssueItem issues PPE items to an employee.
//
// Business rules:
// - issuer (issuedByUserID) must be a user with role "warehouse"
// - issued_to user determines department_id
// - stock must be sufficient; never allow negative inventory
func (s *TransactionsService) IssueItem(itemID string, quantity int, issuedToUserID string, issuedByUserID string) (models.Transaction, error) {
	if issuedToUserID == "" {
		return models.Transaction{}, errors.New("issued_to_user_id is required")
	}
	if issuedByUserID == "" {
		return models.Transaction{}, errors.New("issued_by_user_id is required")
	}
	if quantity <= 0 {
		return models.Transaction{}, errors.New("quantity must be > 0")
	}
	if itemID == "" {
		return models.Transaction{}, errors.New("item_id is required")
	}

	issuer, ok, err := s.usersRepo.GetByID(issuedByUserID)
	if err != nil {
		return models.Transaction{}, err
	}
	if !ok {
		return models.Transaction{}, errors.New("issuer not found")
	}
	if issuer.Role != "warehouse" {
		return models.Transaction{}, errors.New("only warehouse staff can issue items")
	}

	issuedTo, ok, err := s.usersRepo.GetByID(issuedToUserID)
	if err != nil {
		return models.Transaction{}, err
	}
	if !ok {
		return models.Transaction{}, errors.New("issued_to user not found")
	}

	item, ok, err := s.itemsRepo.GetByID(itemID)
	if err != nil {
		return models.Transaction{}, err
	}
	if !ok {
		return models.Transaction{}, errors.New("item not found")
	}
	if quantity > item.Quantity {
		return models.Transaction{}, errors.New(
			// Error format is stable for UX; easy to parse/log later if needed.
			// Example: Not enough stock. Available: 12, requested: 20
			fmt.Sprintf("Not enough stock. Available: %d, requested: %d", item.Quantity, quantity),
		)
	}

	// Decrease stock (repository also guards against negatives).
	if _, err := s.itemsRepo.DecreaseQuantityIfEnough(itemID, quantity); err != nil {
		return models.Transaction{}, err
	}

	tx := models.Transaction{
		ID:             NewID("TX"),
		ItemID:         itemID,
		ItemName:       item.Name,
		Quantity:       quantity,
		IssuedToUserID: issuedToUserID,
		IssuedByUserID: issuedByUserID,
		DepartmentID:   issuedTo.DepartmentID,
		Timestamp:      NowTimestamp(),
	}
	if err := s.transactionsRepo.Add(tx); err != nil {
		return models.Transaction{}, err
	}
	return tx, nil
}

// ReturnItem records a return against an existing issue transaction and updates stock.
//
// Business rules:
// - the referenced transaction must exist
// - total returned (summed across returns) cannot exceed original issued quantity
// - receivedByUserID must have role "warehouse"
func (s *TransactionsService) ReturnItem(
	transactionID string,
	quantity int,
	returnedByUserID string,
	receivedByUserID string,
) (models.Return, error) {
	if transactionID == "" {
		return models.Return{}, errors.New("transaction_id is required")
	}
	if quantity <= 0 {
		return models.Return{}, errors.New("quantity must be > 0")
	}
	if returnedByUserID == "" {
		return models.Return{}, errors.New("returned_by_user_id is required")
	}
	if receivedByUserID == "" {
		return models.Return{}, errors.New("received_by_user_id is required")
	}

	issueTx, ok, err := s.transactionsRepo.GetByID(transactionID)
	if err != nil {
		return models.Return{}, err
	}
	if !ok {
		return models.Return{}, errors.New("transaction not found")
	}

	receivedByUser, ok, err := s.usersRepo.GetByID(receivedByUserID)
	if err != nil {
		return models.Return{}, err
	}
	if !ok {
		return models.Return{}, errors.New("received_by_user not found")
	}
	if receivedByUser.Role != "warehouse" {
		return models.Return{}, errors.New(`received_by_user must have role "warehouse"`)
	}

	// Validate against partial returns.
	returns, err := s.returnsRepo.ListByTransactionID(transactionID)
	if err != nil {
		return models.Return{}, err
	}
	totalReturned := 0
	for _, r := range returns {
		totalReturned += r.QuantityReturned
	}
	if totalReturned+quantity > issueTx.Quantity {
		return models.Return{}, errors.New("return quantity exceeds issued quantity")
	}

	// Update stock.
	if err := s.itemsRepo.IncreaseQuantity(issueTx.ItemID, quantity); err != nil {
		return models.Return{}, err
	}

	ret := models.Return{
		ID:                NewID("RET"),
		TransactionID:    transactionID,
		ItemID:            issueTx.ItemID,
		QuantityReturned: quantity,
		ReturnedByUserID: returnedByUserID,
		ReceivedByUserID: receivedByUserID,
		DepartmentID:      issueTx.DepartmentID,
		Timestamp:         NowTimestamp(),
	}
	if err := s.returnsRepo.Add(ret); err != nil {
		return models.Return{}, err
	}
	return ret, nil
}

