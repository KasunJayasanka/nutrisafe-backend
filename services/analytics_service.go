package services

import (
	"context"
	"errors"
	"math"
	"time"

	"backend/models"

	"gorm.io/gorm"
)

type AnalyticsService struct{ db *gorm.DB }

func NewAnalyticsService(db *gorm.DB) *AnalyticsService { return &AnalyticsService{db: db} }

// ---------- Summary ----------

type NutrAvg struct {
	AvgConsumed float64 `json:"avg_consumed"`
	AvgGoal     float64 `json:"avg_goal,omitempty"`
	AvgPercent  float64 `json:"avg_percent,omitempty"`
	Unit        string  `json:"unit,omitempty"`
}

type AnalyticsSummary struct {
	Range struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"range"`

	Macros map[string]NutrAvg `json:"macros"` // calories, protein, carbs, fat
	Micros map[string]NutrAvg `json:"micros"` // sodium, sugar (extend when you store more)
	Other  map[string]NutrAvg `json:"other"`  // hydration, exercise

	Safety struct {
		ScorePct     float64 `json:"score_pct"`
		TotalItems   int64   `json:"total_items,omitempty"`
		UnsafeItems  int64   `json:"unsafe_items,omitempty"`
		SafeItems    int64   `json:"safe_items,omitempty"`
		UnknownItems int64   `json:"unknown_items,omitempty"`
	} `json:"safety"`

	Metadata struct {
		DaysCounted        int  `json:"days_counted"`
		IncludeMissingDays bool `json:"include_missing_days"`
	} `json:"metadata"`
}

func (s *AnalyticsService) Summary(
	ctx context.Context, userID uint, from, to time.Time, includeMissing bool,
) (*AnalyticsSummary, error) {

	var rows []models.DailyProgress
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND date BETWEEN ? AND ?", userID, dayStart(from), dayEnd(to)).
		Order("date ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	goal, err := s.getGoalSnapshot(ctx, userID, time.Now())
	if err != nil {
		return nil, err
	}

	// index rows by yyyy-mm-dd for missing-day handling
	idx := map[string]models.DailyProgress{}
	for _, r := range rows {
		idx[r.Date.Format("2006-01-02")] = r
	}

	// accumulator
	type acc struct{ sum, gsum, psum float64 }
	m := map[string]*acc{
		"calories": {}, "protein": {}, "carbs": {}, "fat": {},
		"sodium": {}, "sugar": {},
		"hydration": {}, "exercise": {},
	}

	var dates []time.Time
	if includeMissing {
		for d := dayStart(from); !d.After(to); d = d.AddDate(0, 0, 1) {
			dates = append(dates, d)
		}
	} else {
		for _, r := range rows {
			dates = append(dates, dayStart(r.Date))
		}
	}
	den := len(dates)
	if den == 0 {
		den = 1 // avoid div by zero; will return zeros
	}

	for _, d := range dates {
		key := d.Format("2006-01-02")
		dp := idx[key] // zero value if not found

		// consumed sums
		m["calories"].sum += dp.Calories
		m["protein"].sum += dp.Protein
		m["carbs"].sum += dp.Carbs
		m["fat"].sum += dp.Fat
		m["sodium"].sum += dp.Sodium
		m["sugar"].sum += dp.Sugar
		m["hydration"].sum += dp.Hydration
		m["exercise"].sum += dp.Exercise

		// goal & daily percent (only where goal>0)
		type pair struct{ g float64; k string; c float64 }
		for _, p := range []pair{
			{goal.Calories, "calories", dp.Calories},
			{goal.Protein, "protein", dp.Protein},
			{goal.Carbs, "carbs", dp.Carbs},
			{goal.Fat, "fat", dp.Fat},
			{goal.Sodium, "sodium", dp.Sodium},
			{goal.Sugar, "sugar", dp.Sugar},
			{goal.Hydration, "hydration", dp.Hydration},
			{goal.Exercise, "exercise", dp.Exercise},
		} {
			m[p.k].gsum += p.g
			if p.g > 0 {
				m[p.k].psum += (p.c / p.g) * 100.0
			}
		}
	}

	// -------- Unbiased safety score (safe / unsafe / unknown + smoothing) --------
	br, err := s.countMealSafetyBreakdown(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	const unknownWeight = 0.5       // treat unknown as half-safe (tune 0.0..1.0)
	const priorAlpha = 1.0          // Beta prior α (pseudo safe)
	const priorBeta = 1.0           // Beta prior β (pseudo unsafe)
	score := computeSafetyScore(br, unknownWeight, priorAlpha, priorBeta)

	out := &AnalyticsSummary{}
	out.Range.From = from.Format("2006-01-02")
	out.Range.To = to.Format("2006-01-02")
	out.Metadata.DaysCounted = len(dates)
	out.Metadata.IncludeMissingDays = includeMissing

	out.Macros = map[string]NutrAvg{
		"calories": {AvgConsumed: avg(m["calories"].sum, len(dates)), AvgGoal: avg(m["calories"].gsum, len(dates)), AvgPercent: avg(m["calories"].psum, len(dates)), Unit: "kcal"},
		"protein":  {AvgConsumed: avg(m["protein"].sum, len(dates)),  AvgGoal: avg(m["protein"].gsum, len(dates)),  AvgPercent: avg(m["protein"].psum, len(dates)),  Unit: "g"},
		"carbs":    {AvgConsumed: avg(m["carbs"].sum, len(dates)),    AvgGoal: avg(m["carbs"].gsum, len(dates)),    AvgPercent: avg(m["carbs"].psum, len(dates)),    Unit: "g"},
		"fat":      {AvgConsumed: avg(m["fat"].sum, len(dates)),      AvgGoal: avg(m["fat"].gsum, len(dates)),      AvgPercent: avg(m["fat"].psum, len(dates)),      Unit: "g"},
	}
	out.Micros = map[string]NutrAvg{
		"sodium": {AvgConsumed: avg(m["sodium"].sum, len(dates)), AvgGoal: avg(m["sodium"].gsum, len(dates)), AvgPercent: avg(m["sodium"].psum, len(dates)), Unit: "mg"},
		"sugar":  {AvgConsumed: avg(m["sugar"].sum, len(dates)),  AvgGoal: avg(m["sugar"].gsum, len(dates)),  AvgPercent: avg(m["sugar"].psum, len(dates)),  Unit: "g"},
	}
	out.Other = map[string]NutrAvg{
		"hydration": {AvgConsumed: avg(m["hydration"].sum, len(dates)), AvgGoal: avg(m["hydration"].gsum, len(dates)), AvgPercent: avg(m["hydration"].psum, len(dates)), Unit: "glasses"},
		"exercise":  {AvgConsumed: avg(m["exercise"].sum, len(dates)),  AvgGoal: avg(m["exercise"].gsum, len(dates)),  AvgPercent: avg(m["exercise"].psum, len(dates)),  Unit: "minutes"},
	}

	out.Safety.ScorePct = score
	out.Safety.TotalItems = br.Total
	out.Safety.UnsafeItems = br.Unsafe
	out.Safety.SafeItems = br.Safe
	out.Safety.UnknownItems = br.Unknown

	return out, nil
}

// ---------- Weekly Overview ----------

type WeeklyOverviewResponse struct {
	WeekStart string `json:"week_start"`
	Mode      string `json:"mode"` // chart|detailed
	Days      any    `json:"days"`
}

type DayChart struct {
	Date        string             `json:"date"`
	Percentages map[string]float64 `json:"percentages"`
}
type Metric struct {
	Actual  float64 `json:"actual"`
	Target  float64 `json:"target"`
	Percent float64 `json:"percent"`
}
type DayDetailed struct {
	Date    string            `json:"date"`
	Metrics map[string]Metric `json:"metrics"`
}

func (s *AnalyticsService) WeeklyOverview(
	ctx context.Context, userID uint, weekStart time.Time, mode string,
) (*WeeklyOverviewResponse, error) {

	if mode != "chart" && mode != "detailed" {
		return nil, errors.New("mode must be 'chart' or 'detailed'")
	}

	from := dayStart(weekStart)
	to := from.AddDate(0, 0, 6)

	var rows []models.DailyProgress
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND date BETWEEN ? AND ?", userID, from, dayEnd(to)).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	idx := map[string]models.DailyProgress{}
	for _, r := range rows {
		idx[r.Date.Format("2006-01-02")] = r
	}

	goal, err := s.getGoalSnapshot(ctx, userID, time.Now())
	if err != nil {
		return nil, err
	}

	out := &WeeklyOverviewResponse{
		WeekStart: from.Format("2006-01-02"),
		Mode:      mode,
	}

	if mode == "chart" {
		var days []DayChart
		for i := 0; i < 7; i++ {
			d := from.AddDate(0, 0, i)
			key := d.Format("2006-01-02")
			dp := idx[key]
			days = append(days, DayChart{
				Date: key,
				Percentages: map[string]float64{
					"calories":      pct(dp.Calories, goal.Calories),
					"protein":       pct(dp.Protein, goal.Protein),
					"carbohydrates": pct(dp.Carbs, goal.Carbs),
					"fat":           pct(dp.Fat, goal.Fat),
					"sodium":        pct(dp.Sodium, goal.Sodium),
					"sugar":         pct(dp.Sugar, goal.Sugar),
					"hydration":     pct(dp.Hydration, goal.Hydration),
					"exercise":      pct(dp.Exercise, goal.Exercise),
				},
			})
		}
		out.Days = days
		return out, nil
	}

	var days []DayDetailed
	for i := 0; i < 7; i++ {
		d := from.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		dp := idx[key]
		days = append(days, DayDetailed{
			Date: key,
			Metrics: map[string]Metric{
				"calories":        {Actual: round2(dp.Calories),  Target: round2(goal.Calories),  Percent: pct(dp.Calories, goal.Calories)},
				"protein_g":       {Actual: round2(dp.Protein),   Target: round2(goal.Protein),   Percent: pct(dp.Protein, goal.Protein)},
				"carbs_g":         {Actual: round2(dp.Carbs),     Target: round2(goal.Carbs),     Percent: pct(dp.Carbs, goal.Carbs)},
				"fat_g":           {Actual: round2(dp.Fat),       Target: round2(goal.Fat),       Percent: pct(dp.Fat, goal.Fat)},
				"sodium_mg":       {Actual: round2(dp.Sodium),    Target: round2(goal.Sodium),    Percent: pct(dp.Sodium, goal.Sodium)},
				"sugar_g":         {Actual: round2(dp.Sugar),     Target: round2(goal.Sugar),     Percent: pct(dp.Sugar, goal.Sugar)},
				"hydration":       {Actual: round2(dp.Hydration), Target: round2(goal.Hydration), Percent: pct(dp.Hydration, goal.Hydration)},
				"exercise_minute": {Actual: round2(dp.Exercise),  Target: round2(goal.Exercise),  Percent: pct(dp.Exercise, goal.Exercise)},
			},
		})
	}
	out.Days = days
	return out, nil
}

// ---------- Safety helpers ----------

type SafetyBreakdown struct {
	Safe    int64 `json:"safe"`
	Unsafe  int64 `json:"unsafe"`
	Unknown int64 `json:"unknown"`
	Total   int64 `json:"total"`
}

func (s *AnalyticsService) countMealSafetyBreakdown(ctx context.Context, userID uint, from, to time.Time) (SafetyBreakdown, error) {
	base := s.db.WithContext(ctx).
		Model(&models.MealItem{}).
		Joins("JOIN meals ON meals.id = meal_items.meal_id").
		Where("meals.user_id = ? AND meals.ate_at BETWEEN ? AND ?", userID, dayStart(from), dayEnd(to))

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return SafetyBreakdown{}, err
	}

	var safe int64
	if err := base.Where("meal_items.safe = ?", true).Count(&safe).Error; err != nil {
		return SafetyBreakdown{}, err
	}

	// Unsafe: explicitly flagged (warnings non-empty) and safe=false
	var unsafe int64
	if err := base.
		Where("meal_items.safe = ? AND COALESCE(meal_items.warnings, '') <> ''", false).
		Count(&unsafe).Error; err != nil {
		return SafetyBreakdown{}, err
	}

	// Unknown: not marked safe and no warnings
	var unknown int64
	if err := base.
		Where("meal_items.safe = ? AND COALESCE(meal_items.warnings, '') = ''", false).
		Count(&unknown).Error; err != nil {
		return SafetyBreakdown{}, err
	}

	return SafetyBreakdown{Safe: safe, Unsafe: unsafe, Unknown: unknown, Total: total}, nil
}

// unknownWeight in [0..1] (0=ignore unknowns, 1=treat as safe)
// alpha,beta are Beta prior params for smoothing small samples.
func computeSafetyScore(br SafetyBreakdown, unknownWeight, alpha, beta float64) float64 {
	safeEff := float64(br.Safe) + unknownWeight*float64(br.Unknown) + alpha
	totalEff := float64(br.Safe+br.Unsafe) + unknownWeight*float64(br.Unknown) + alpha + beta
	if totalEff <= 0 {
		return 100.0
	}
	return round2((safeEff / totalEff) * 100.0)
}

// ---------- internals ----------

func (s *AnalyticsService) getGoalSnapshot(ctx context.Context, userID uint, _ time.Time) (*models.DailyGoal, error) {
	var g models.DailyGoal
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		First(&g).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &models.DailyGoal{}, nil
		}
		return nil, err
	}
	return &g, nil
}

func pct(actual, goal float64) float64 {
	if goal <= 0 {
		if actual <= 0 {
			return 0
		}
		return 100
	}
	return round2((actual / goal) * 100.0)
}

func avg(sum float64, n int) float64 {
	if n <= 0 {
		return 0
	}
	return round2(sum / float64(n))
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func dayStart(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()) }
func dayEnd(t time.Time) time.Time   { return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), t.Location()) }
