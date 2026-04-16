package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DashboardRepository aggregates read models for company and affiliate UIs.
type DashboardRepository struct {
	pool *pgxpool.Pool
}

func NewDashboardRepository(pool *pgxpool.Pool) *DashboardRepository {
	return &DashboardRepository{pool: pool}
}

// CompanySummary is performance for one merchant campain (your "program" / campaign scope).
type CompanySummary struct {
	CampainID            uuid.UUID `json:"campain_id"`
	CampainName          string    `json:"campain_name"`
	OrderCount           int64     `json:"order_count"`
	SalesTotalCents      int64     `json:"sales_total_cents"`
	CommissionsPending   int64     `json:"commissions_pending_cents"`
	CommissionsApproved  int64     `json:"commissions_approved_cents"`
	CommissionsPaid      int64     `json:"commissions_paid_cents"`
	ActiveAffiliateCount int64     `json:"active_affiliate_count"`
}

// AffiliateLeaderboardRow is top affiliates by commission volume in the campain.
type AffiliateLeaderboardRow struct {
	AffiliateID   uuid.UUID `json:"affiliate_id"`
	Code          string    `json:"code"`
	UserID        string    `json:"user_id"`
	CommissionSum int64     `json:"commission_total_cents"`
	OrderCount    int64     `json:"attributed_orders"`
}

// CompanySummaryWithLeaders returns campain KPIs and top performers.
func (r *DashboardRepository) CompanySummaryWithLeaders(ctx context.Context, campainID uuid.UUID, topN int) (*CompanySummary, []AffiliateLeaderboardRow, error) {
	var name string
	err := r.pool.QueryRow(ctx, `SELECT name FROM campains WHERE id = $1`, campainID).Scan(&name)
	if err != nil {
		return nil, nil, err
	}

	var orderCount, sales int64
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total_cents), 0) FROM orders WHERE campain_id = $1
	`, campainID).Scan(&orderCount, &sales)
	if err != nil {
		return nil, nil, err
	}

	var pend, appr, paid int64
	rows, err := r.pool.Query(ctx, `
		SELECT c.status, COALESCE(SUM(c.amount_cents), 0)
		FROM commissions c
		JOIN affiliates a ON a.id = c.affiliate_id
		WHERE a.campain_id = $1
		GROUP BY c.status
	`, campainID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var sum int64
		if err := rows.Scan(&st, &sum); err != nil {
			return nil, nil, err
		}
		switch st {
		case "pending":
			pend = sum
		case "approved":
			appr = sum
		case "paid":
			paid = sum
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var affCount int64
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM affiliates WHERE campain_id = $1 AND status = 'active'
	`, campainID).Scan(&affCount)
	if err != nil {
		return nil, nil, err
	}

	summary := &CompanySummary{
		CampainID:            campainID,
		CampainName:          name,
		OrderCount:           orderCount,
		SalesTotalCents:      sales,
		CommissionsPending:   pend,
		CommissionsApproved:  appr,
		CommissionsPaid:      paid,
		ActiveAffiliateCount: affCount,
	}

	if topN <= 0 {
		topN = 10
	}
	lr, err := r.pool.Query(ctx, `
		SELECT a.id, a.code, a.user_id,
			COALESCE(SUM(c.amount_cents), 0),
			COUNT(DISTINCT c.order_id)
		FROM affiliates a
		LEFT JOIN commissions c ON c.affiliate_id = a.id
		WHERE a.campain_id = $1 AND a.status = 'active'
		GROUP BY a.id, a.code, a.user_id
		ORDER BY COALESCE(SUM(c.amount_cents), 0) DESC
		LIMIT $2
	`, campainID, topN)
	if err != nil {
		return nil, nil, err
	}
	defer lr.Close()
	var leaders []AffiliateLeaderboardRow
	for lr.Next() {
		var row AffiliateLeaderboardRow
		if err := lr.Scan(&row.AffiliateID, &row.Code, &row.UserID, &row.CommissionSum, &row.OrderCount); err != nil {
			return nil, nil, err
		}
		leaders = append(leaders, row)
	}
	if err := lr.Err(); err != nil {
		return nil, nil, err
	}
	if leaders == nil {
		leaders = []AffiliateLeaderboardRow{}
	}

	return summary, leaders, nil
}

// AffiliateProgramStat is earnings per merchant program (campain) the affiliate participates in.
type AffiliateProgramStat struct {
	CampainID        uuid.UUID `json:"campain_id"`
	CampainName      string    `json:"campain_name"`
	AccruedCents     int64     `json:"accrued_cents"` // pending + approved (not yet paid out)
	PaidCents        int64     `json:"paid_cents"`
	OrderCount       int64     `json:"attributed_orders"`
	ReferralCode     string    `json:"referral_code"`
}

// AffiliateProgramStats lists performance per program for a user.
func (r *DashboardRepository) AffiliateProgramStats(ctx context.Context, userID string) ([]AffiliateProgramStat, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			o.id,
			o.name,
			a.code,
			COALESCE(SUM(CASE WHEN c.status IN ('pending', 'approved') THEN c.amount_cents ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN c.status = 'paid' THEN c.amount_cents ELSE 0 END), 0),
			COUNT(DISTINCT c.order_id)
		FROM affiliates a
		INNER JOIN campains o ON o.id = a.campain_id
		LEFT JOIN commissions c ON c.affiliate_id = a.id
		WHERE a.user_id = $1 AND a.status = 'active'
		GROUP BY o.id, o.name, a.code
		ORDER BY o.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AffiliateProgramStat
	for rows.Next() {
		var s AffiliateProgramStat
		if err := rows.Scan(
			&s.CampainID,
			&s.CampainName,
			&s.ReferralCode,
			&s.AccruedCents,
			&s.PaidCents,
			&s.OrderCount,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if out == nil {
		out = []AffiliateProgramStat{}
	}
	return out, rows.Err()
}
