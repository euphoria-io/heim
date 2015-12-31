package psql

type Nick struct {
	Room   string
	UserID string `db:"user_id"`
	Nick   string
}
