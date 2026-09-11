package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"ai-operations-copilot/internal/config"

	_ "modernc.org/sqlite"
)

func main() {
	cfg := config.Load()
	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o755); err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	migration, err := os.ReadFile("migrations/001_initial.sql")
	if err != nil {
		log.Fatal(err)
	}
	if _, err = db.Exec(string(migration)); err != nil {
		log.Fatal(err)
	}
	if _, err = db.Exec("DELETE FROM deliveries; DELETE FROM payments; DELETE FROM orders; DELETE FROM customers;"); err != nil {
		log.Fatal(err)
	}
	rng := rand.New(rand.NewSource(42))
	now := time.Now().UTC().Truncate(time.Second)
	exampleOrderIDs := []string{"4521", "1289", "2231", "1045", "1201"}
	for i := 1; i <= 100; i++ {
		_, err = db.Exec(`INSERT INTO customers(id,name,email,phone,created_at) VALUES(?,?,?,?,?)`, fmt.Sprintf("C%04d", i), fmt.Sprintf("Customer %d", i), fmt.Sprintf("customer%03d@example.com", i), fmt.Sprintf("+919900%06d", i), now.Add(-time.Duration(rng.Intn(365))*24*time.Hour))
		if err != nil {
			log.Fatal(err)
		}
	}
	for i := 1; i <= 300; i++ {
		orderID := fmt.Sprintf("%d", 4000+i)
		if i <= len(exampleOrderIDs) {
			orderID = exampleOrderIDs[i-1]
		}
		customerID := fmt.Sprintf("C%04d", 1+rng.Intn(100))
		amount := int64(10000 + rng.Intn(90000))
		orderStatus, paymentStatus, deliveryStatus := "confirmed", "paid", "scheduled"
		var rejection any
		switch i % 7 {
		case 0:
			orderStatus, paymentStatus, deliveryStatus = "pending", "pending", "not_scheduled"
		case 1:
			orderStatus, paymentStatus, deliveryStatus = "pending", "failed", "not_scheduled"
		case 2:
			deliveryStatus = "not_scheduled"
		case 3:
			deliveryStatus = "delayed"
		case 4:
			orderStatus, paymentStatus, deliveryStatus = "cancelled", "refunded", "cancelled"
		case 5:
			orderStatus, paymentStatus, deliveryStatus, rejection = "rejected", "failed", "not_scheduled", "vehicle verification failed"
		}
		created := now.Add(-time.Duration(rng.Intn(30)) * 24 * time.Hour)
		_, err = db.Exec(`INSERT INTO orders(id,customer_id,status,total_amount,currency,rejection_reason,created_at) VALUES(?,?,?,?,?,?,?)`, orderID, customerID, orderStatus, amount, "INR", rejection, created)
		if err != nil {
			log.Fatal(err)
		}
		paidAt := any(nil)
		txn := any(nil)
		if paymentStatus == "paid" || paymentStatus == "refunded" {
			paidAt, txn = created.Add(time.Hour), fmt.Sprintf("TXN%06d", i)
		}
		_, err = db.Exec(`INSERT INTO payments(id,order_id,status,amount,transaction_id,paid_at) VALUES(?,?,?,?,?,?)`, fmt.Sprintf("P%04d", i), orderID, paymentStatus, amount, txn, paidAt)
		if err != nil {
			log.Fatal(err)
		}
		scheduled, delivered := any(nil), any(nil)
		courier, tracking := any(nil), any(nil)
		if deliveryStatus == "scheduled" || deliveryStatus == "delivered" || deliveryStatus == "delayed" {
			scheduled, courier, tracking = created.Add(48*time.Hour), "Swift Logistics", fmt.Sprintf("TRK%06d", i)
		}
		if i%6 == 0 && deliveryStatus == "scheduled" {
			deliveryStatus, delivered = "delivered", created.Add(96*time.Hour)
		}
		_, err = db.Exec(`INSERT INTO deliveries(id,order_id,status,scheduled_at,delivered_at,courier,tracking_id) VALUES(?,?,?,?,?,?,?)`, fmt.Sprintf("D%04d", i), orderID, deliveryStatus, scheduled, delivered, courier, tracking)
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("seeded 100 customers and 300 orders into %s", cfg.DatabasePath)
}
