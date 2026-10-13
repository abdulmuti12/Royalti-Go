package application

import (
	"context"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/domain"
)

// Transactions are scoped through orders.customer_id. The Laravel source
// mistakenly compares order_id to a customer UUID; preserving that bug would
// expose or hide records. The HTTP paths and payload keys remain unchanged.
func (s *Service) transactions(ctx context.Context, c domain.Command) (domain.Result, error) {
	orders, e := s.Store.List(ctx, "orders", customerQuery(c.Principal.ID))
	if e != nil {
		return domain.Result{}, e
	}
	q := domain.Q(domain.In("order_id", listIDs(orders, "id")))
	if c.Action == "customer-transactions" {
		q.Sort = []domain.Sort{{Field: "created_at", Desc: true}}
		rows, e := s.Store.List(ctx, "transaction", q)
		return domain.Success(rows, "Data transaksi berhasil diambil"), e
	}
	if c.Action == "customer-transaction" {
		q.Filters = append(q.Filters, domain.Equal("id", c.ID))
		row, e := s.Store.Find(ctx, "transaction", q)
		return domain.Success(row, "Detail transaksi berhasil diambil"), missing(e, "Not Found", "Transaksi tidak ditemukan atau bukan milik Anda")
	}
	if c.Action == "delete-transaction" {
		q.Filters = append(q.Filters, domain.Equal("id", c.ID))
		row, e := s.Store.Find(ctx, "transaction", q)
		if e != nil {
			return domain.Result{}, missing(e, "Not Found", "Transaction not found or access denied")
		}
		if domain.Text(row["status"]) == "success" {
			return domain.Result{}, domain.Fail("Forbidden", "Transaksi berhasil tidak dapat dihapus customer")
		}
		e = s.Store.Delete(ctx, "transaction", q)
		return domain.Success(nil, "Transaksi berhasil dihapus"), e
	}
	rules := map[string]string{"order_id": "required|integer|min:1|exists:orders,id", "total_payment": "required|integer|min:0", "status": "required|in:pending,success,failed,cancelled", "note": "nullable|string", "address_id": "required|integer|min:1|exists:addresses,id"}
	if c.Action == "update-transaction" {
		delete(rules, "order_id")
		for k, v := range rules {
			if v[:8] == "required" {
				rules[k] = "sometimes" + v[8:]
			}
		}
	}
	if e := s.validate(ctx, c.Input, rules, ""); e != nil {
		return domain.Result{}, e
	}
	if domain.Text(c.Input["status"]) == "success" {
		return domain.Result{}, domain.Fail("Forbidden", "Status success hanya dapat dikonfirmasi admin atau payment provider")
	}
	data := domain.Record{}
	for key := range rules {
		if v, ok := c.Input[key]; ok {
			data[key] = v
		}
	}
	var row domain.Record
	e = s.Store.Transaction(ctx, func(db domain.Store) error {
		if e := s.lockCustomer(ctx, db, c.Principal.ID); e != nil {
			return e
		}
		if c.Action == "create-transaction" {
			if _, e := db.Find(ctx, "orders", domain.Q(domain.Equal("id", data["order_id"]), domain.Equal("customer_id", c.Principal.ID))); e != nil {
				return missing(e, "Forbidden", "Order bukan milik Anda")
			}
		} else {
			q.Filters = append(q.Filters, domain.Equal("id", c.ID))
			var e error
			row, e = db.Find(ctx, "transaction", q)
			if e != nil {
				return missing(e, "Not Found", "Transaction not found or access denied")
			}
			if domain.Text(row["status"]) == "success" {
				return domain.Fail("Forbidden", "Transaksi berhasil tidak dapat diubah customer")
			}
		}
		if data["address_id"] != nil {
			if _, e := db.Find(ctx, "addresses", domain.Q(domain.Equal("id", data["address_id"]), domain.Equal("user_id", c.Principal.ID))); e != nil {
				return missing(e, "Forbidden", "Alamat bukan milik Anda")
			}
		}
		var e error
		if c.Action == "create-transaction" {
			token, e := domain.RandomToken()
			if e != nil {
				return e
			}
			data["transaction_code"] = "TRX-" + time.Now().Format("20060102") + "-" + token[:8]
			row, e = db.Insert(ctx, "transaction", data)
		} else {
			e = db.Update(ctx, "transaction", q, data)
			if e == nil {
				row, e = db.Find(ctx, "transaction", q)
			}
		}
		return e
	})
	msg := "Transaksi berhasil ditambahkan"
	if c.Action == "update-transaction" {
		msg = "Transaksi berhasil diperbarui"
	}
	return domain.Success(row, msg), e
}
