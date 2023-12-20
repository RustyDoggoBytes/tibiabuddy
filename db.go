package main

import (
	"database/sql"
	"errors"

	_ "modernc.org/sqlite"
)

type repositoryClient struct {
	Db *sql.DB
}

func RepositoryClient(filepath string) (*repositoryClient, error) {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS former_names (id INTEGER PRIMARY KEY, name TEXT, notification_emails TEXT, last_checked DATETIME, last_updated_status DATETIME, status TEXT)")
	if err != nil {
		return nil, err
	}

	return &repositoryClient{Db: db}, nil
}

func (r *repositoryClient) Close() {
	r.Db.Close()
}

func (r *repositoryClient) GetFormerNames() ([]FormerName, error) {
	rows, err := r.Db.Query("SELECT name, notification_emails, last_checked, last_updated_status, status FROM former_names")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var formerNames []FormerName

	for rows.Next() {
		var fn FormerName
		err := rows.Scan(&fn.Name, &fn.NotificationEmail, &fn.LastChecked, &fn.LastUpdatedStatus, &fn.Status)
		if err != nil {
			return nil, err
		}
		formerNames = append(formerNames, fn)
	}

	return formerNames, nil
}

func (r *repositoryClient) SaveFormerName(fn FormerName) error {
	_, err := r.Db.Exec("INSERT OR REPLACE INTO former_names (id, name, notification_emails, last_checked, last_updated_status, status) VALUES ((SELECT id from former_names where  name = ?), ?, ?, ?, ?, ?)", fn.Name, fn.Name, fn.NotificationEmail, fn.LastChecked, fn.LastUpdatedStatus, fn.Status)

	return err
}

func (r *repositoryClient) DeleteFormerName(name string) error {
	result, err := r.Db.Exec("DELETE FROM former_names where name = ?", name)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("not found")
	}

	return err

}
