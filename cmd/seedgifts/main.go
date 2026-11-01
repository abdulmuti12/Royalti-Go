package main

import (
	"context"
	"flag"
	"log"

	"github.com/abdulmuti12/api-point-go/internal/domain"
	"github.com/abdulmuti12/api-point-go/internal/platform"
	"github.com/abdulmuti12/api-point-go/internal/repository/gormrepo"
)

// seedgifts upserts the demo reward-gift catalogue and tops up the demo
// customer's active points so at least one card shows the "Redeem" state.
//
// Usage:
//
//	go run ./cmd/seedgifts [-email customer@example.com]
//
// Idempotent: gifts are matched by name, the customer's points by email.

// demoGift is one row of the demo catalogue. The 6 entries mirror the
// "Featured Rewards" mockup. image/image2/image3 are relative to the API's
// public storage disk (storage/app/public) and are resolved to absolute URLs
// by the app's Asset() helper at read time.
type demoGift struct {
	name     string
	pts      int64
	featured bool
	desc     string
	image    string
	image2   string
	image3   string
}

var demoGifts = []demoGift{
	{
		name:     "Exclusive Trip to Italy",
		pts:      350000,
		featured: true,
		image:    "gifts/trip-italy.png",
		image2:   "gifts/trip-italy-2.png",
		desc: "A week in an exclusive villa overlooking Lake Como with private chef and guided city tours of Milan and Florence.\n\nPackage Includes:\n- 7-night villa stay in Como\n- Private transfer to Milan\n- Guided Florence walking tour\n- Wine tasting session\n- Personal concierge line",
	},
	{
		name:     "iPhone 18 Pro Max",
		pts:      200000,
		featured: true,
		image:    "gifts/iphone-18.png",
		image2:   "gifts/iphone-18-2.png",
		desc: "The latest flagship iPhone in titanium with the most advanced camera system yet.\n\nPackage Includes:\n- iPhone 18 Pro Max, 1TB\n- Free shipping\n- 12-month device warranty\n- Setup service",
	},
	{
		name:     "Diamond Jewelry Set",
		pts:      180000,
		featured: false,
		image:    "gifts/diamond-jewelry.png",
		desc: "A hand-set diamond necklace and earrings in 18k white gold, crafted by master jewelers.\n\nPackage Includes:\n- Diamond necklace\n- Diamond earrings\n- 18k white gold setting\n- Certificate of authenticity\n- Complimentary engraving",
	},
	{
		name:     "Business Class Getaway",
		pts:      150000,
		featured: false,
		image:    "gifts/business-class.png",
		desc: "Round-trip business class to your destination of choice with lounge access at departure.\n\nPackage Includes:\n- Business class round-trip\n- Airport lounge access\n- 23kg checked baggage\n- Seat selection\n- 24h support line",
	},
	{
		name:     "Fine Porcelain Tableware",
		pts:      120000,
		featured: false,
		image:    "gifts/porcelain-tableware.png",
		desc: "A hand-painted fine porcelain service for a full dining set, oven and dishwasher safe.\n\nPackage Includes:\n- 12 place settings\n- Dinner and dessert plates\n- Hand-painted pattern\n- Dishwasher safe\n- Presentation box",
	},
	{
		name:     "Fine Dining for Two",
		pts:      100000,
		featured: false,
		image:    "gifts/fine-dining.png",
		desc: "A three-course tasting menu for two at a michelin-listed restaurant, with wine pairing.\n\nPackage Includes:\n- 3-course tasting menu\n- Wine pairing for two\n- Chef's table seating\n- Complimentary aperitivo\n- Reservation guarantee",
	},
}

func main() {
	email := flag.String("email", "customer@example.com", "demo customer email to top up")
	pts := flag.Int64("points", 350000, "active points to grant the demo customer")
	flag.Parse()

	c, e := platform.LoadConfig()
	if e != nil {
		log.Fatal(e)
	}
	db, e := platform.OpenDatabase(c)
	if e != nil {
		log.Fatal(e)
	}
	sqlDB, e := db.DB()
	if e != nil {
		log.Fatal(e)
	}
	defer sqlDB.Close()
	store := gormrepo.New(db)
	ctx := context.Background()

	// Reference an existing active point rule; the claims flow only needs
	// the redeem_points row to carry total_point_active.
	pointRule, e := store.Find(ctx, "points", domain.Q(domain.Equal("status", "active")))
	if e != nil {
		log.Printf("warning: no active point rule found; seed will still insert gifts but top-up is skipped: %v", e)
	}

	// Upsert each demo gift by name.
	for _, g := range demoGifts {
		// Build the full record once so insert and update share the same
		// fields. image fields are only written when non-empty so a gift
		// seeded without images is not blanked on a later re-run.
		rec := domain.Record{
			"name":        g.name,
			"total_point": g.pts,
			"description": g.desc,
			"status":      "active",
			"is_featured": g.featured,
		}
		for field, val := range map[string]string{"image": g.image, "image2": g.image2, "image3": g.image3} {
			if val != "" {
				rec[field] = val
			}
		}

		existing, e := store.Find(ctx, "gifts", domain.Q(domain.Equal("name", g.name)))
		if e != nil {
			if e != domain.ErrNotFound {
				log.Fatal(e)
			}
			r, e := store.Insert(ctx, "gifts", rec)
			if e != nil {
				log.Fatalf("insert gift %q: %v", g.name, e)
			}
			log.Printf("inserted gift %q (id %s)", g.name, domain.Text(r["id"]))
			continue
		}
		if e := store.Update(ctx, "gifts", domain.Q(domain.Equal("id", existing["id"])), rec); e != nil {
			log.Fatalf("update gift %q: %v", g.name, e)
		}
		log.Printf("updated gift %q (id %s)", g.name, domain.Text(existing["id"]))
	}

	// Top up the demo customer's active points so the cheapest featured card
	// (Exclusive Trip to Italy at 350k) is reachable.
	if email != nil && *email != "" && pointRule != nil {
		cust, e := store.Find(ctx, "customers", domain.Q(domain.Equal("email", *email)))
		if e != nil {
			log.Printf("customer %q not found; skipping point top-up: %v", *email, e)
		} else {
			rp, e := store.Find(ctx, "redeem_points", domain.Q(domain.Equal("customer_id", cust["id"]), domain.Equal("point_id", pointRule["id"])))
			if e != nil {
				if e == domain.ErrNotFound {
					r, e := store.Insert(ctx, "redeem_points", domain.Record{
						"customer_id":        cust["id"],
						"point_id":           pointRule["id"],
						"total_transaction":  0,
						"total_point_earned": *pts,
						"total_point_active": *pts,
					})
					if e != nil {
						log.Fatalf("insert redeem_points for %q: %v", *email, e)
					}
					log.Printf("granted %d active points to %s (new redeem_points %s)", *pts, *email, domain.Text(r["id"]))
				} else {
					log.Fatalf("find redeem_points for %q: %v", *email, e)
				}
			} else {
				changes := domain.Record{"total_point_active": domain.Integer(rp["total_point_active"]) + *pts}
				if e := store.Update(ctx, "redeem_points", domain.Q(domain.Equal("id", rp["id"])), changes); e != nil {
					log.Fatalf("update redeem_points for %q: %v", *email, e)
				}
				log.Printf("topped up %d active points to %s (total now %d)", *pts, *email, domain.Integer(rp["total_point_active"])+*pts)
			}
		}
	}

	log.Print("Demo reward seeding complete")
}
