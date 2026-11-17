// «Турнир месяца»: последнее воскресенье месяца, 10:00–13:00 по Ташкенту (05:00–08:00 UTC).
// Те же правила — на сервере (rating.TourStart): меняется одно — меняется и другое.

export const TOUR_HOURS = 3

// Начало турнира, окно которого ещё не кончилось к now.
export function tourStart(now: Date): Date {
  for (let add = 0; ; add++) {
    const last = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + add + 1, 0, 5))
    last.setUTCDate(last.getUTCDate() - last.getUTCDay())
    if (now.getTime() < last.getTime() + TOUR_HOURS * 3_600_000) return last
  }
}
