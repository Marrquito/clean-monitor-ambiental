package chat

import (
	"fmt"
	"time"

	"clean-backend/internal/storage"
)

const (
	PeriodLastHour    = "last_hour"
	PeriodLast2Hours  = "last_2_hours"
	PeriodLast24Hours = "last_24_hours"
	PeriodToday       = "today"
	PeriodYesterday   = "yesterday"
	PeriodLast7Days   = "last_7_days"
	PeriodThisWeek    = "this_week"
	PeriodLastWeek    = "last_week"
	PeriodCustom      = "custom"

	maxCustomSpan = 31 * 24 * time.Hour
	dateLayout    = "2006-01-02"
)

type Period struct {
	Name  string
	Label string
	Range storage.Range
}

// ResolvePeriod turns a named or explicit period into a UTC range. Calendar periods
// follow the application timezone (America/Sao_Paulo by default), not the server's.
func ResolvePeriod(name, startDate, endDate string, now time.Time, loc *time.Location) (Period, error) {
	now = now.UTC()
	local := now.In(loc)
	startOfToday := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)

	var from, to time.Time
	var label string
	switch name {
	case PeriodLastHour:
		from, to, label = now.Add(-time.Hour), now, "na última hora"
	case PeriodLast2Hours:
		from, to, label = now.Add(-2*time.Hour), now, "nas últimas 2 horas"
	case PeriodLast24Hours:
		from, to, label = now.Add(-24*time.Hour), now, "nas últimas 24 horas"
	case PeriodToday:
		from, to, label = startOfToday, now, "hoje"
	case PeriodYesterday:
		from, to, label = startOfToday.AddDate(0, 0, -1), startOfToday, "ontem"
	case PeriodLast7Days:
		from, to, label = now.AddDate(0, 0, -7), now, "nos últimos 7 dias"
	case PeriodThisWeek:
		from, to, label = startOfWeek(startOfToday), now, "nesta semana"
	case PeriodLastWeek:
		thisWeek := startOfWeek(startOfToday)
		from, to, label = thisWeek.AddDate(0, 0, -7), thisWeek, "na semana passada"
	case PeriodCustom:
		if startDate == "" {
			return Period{}, fmt.Errorf("não entendi o período da pergunta; use, por exemplo, \"hoje\", \"ontem\", \"esta semana\", \"últimos 7 dias\" ou datas como \"de 20/11/2026 a 22/11/2026\"")
		}
		if endDate == "" {
			endDate = local.Format(dateLayout)
		}
		start, err := time.ParseInLocation(dateLayout, startDate, loc)
		if err != nil {
			return Period{}, fmt.Errorf("data inicial inválida; use o formato AAAA-MM-DD")
		}
		end, err := time.ParseInLocation(dateLayout, endDate, loc)
		if err != nil {
			return Period{}, fmt.Errorf("data final inválida; use o formato AAAA-MM-DD")
		}
		from, to = start, end.AddDate(0, 0, 1)
		if to.After(now) {
			to = now
		}
		if !from.Before(to) {
			return Period{}, fmt.Errorf("o intervalo informado não contém medições possíveis (a data final deve ser igual ou posterior à inicial e não estar no futuro)")
		}
		if to.Sub(from) > maxCustomSpan {
			return Period{}, fmt.Errorf("o intervalo máximo de consulta é de 31 dias")
		}
		label = fmt.Sprintf("de %s a %s", start.Format("02/01/2006"), end.Format("02/01/2006"))
	default:
		return Period{}, fmt.Errorf("período não suportado: %q", name)
	}

	return Period{
		Name:  name,
		Label: label,
		Range: storage.Range{From: from.UTC(), To: to.UTC()},
	}, nil
}

// startOfWeek returns local midnight of the Monday of the week containing day.
func startOfWeek(day time.Time) time.Time {
	offset := (int(day.Weekday()) + 6) % 7 // Monday = 0
	return day.AddDate(0, 0, -offset)
}

// Baseline returns the equal-length window immediately before p, used for comparisons.
func (p Period) Baseline() Period {
	span := p.Range.To.Sub(p.Range.From)
	return Period{
		Name:  p.Name,
		Label: "no período anterior de mesma duração",
		Range: storage.Range{From: p.Range.From.Add(-span), To: p.Range.From},
	}
}
