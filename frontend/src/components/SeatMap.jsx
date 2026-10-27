import { passengerSeats, seatState } from '../utils/seats.js'

const stateLabels = { available: 'Available', selected: 'Selected', booked: 'Booked by someone else', owned: 'Your seat' }

function SeatShape() {
  return <><rect x="7" y="3" width="30" height="12" rx="5" /><rect x="5" y="11" width="34" height="26" rx="8" /><path d="M7 29h30M8 16v9m28-9v9" fill="none" stroke="currentColor" strokeOpacity=".25" strokeWidth="2" /><rect x="2" y="21" width="5" height="15" rx="2" /><rect x="37" y="21" width="5" height="15" rx="2" /></>
}

export default function SeatMap({ layout, selected = [], owned = [], onToggle, disabled = false, compact = false }) {
  const seats = passengerSeats(layout.capacity)
  const rows = 1 + Math.ceil((layout.capacity - 1) / 3)
  const height = 170 + rows * 66
  const summary = seats.map((seat) => `Seat ${seat.number}: ${stateLabels[seatState(seat.id, layout, selected, owned)]}`).join('. ')

  return <div className={`seat-map ${compact ? 'seat-map--compact' : ''}`}>
    <div className="car-top-view" style={{ '--car-height': `${height}px`, aspectRatio: `236 / ${height}` }}>
      <svg className="car-silhouette" viewBox={`0 0 236 ${height}`} role={compact ? 'img' : undefined} aria-label={compact ? `Top view of car. Driver at front left. ${summary}` : undefined} aria-hidden={compact ? undefined : true}>
        <g fill="var(--seat-car-trim)"><rect x="13" y="74" width="16" height="46" rx="7" /><rect x="207" y="74" width="16" height="46" rx="7" /><rect x="13" y={height - 110} width="16" height="46" rx="7" /><rect x="207" y={height - 110} width="16" height="46" rx="7" /></g>
        <path d={`M68 13 Q118 -1 168 13 Q205 23 210 74 L210 ${height - 61} Q206 ${height - 17} 168 ${height - 12} L68 ${height - 12} Q30 ${height - 17} 26 ${height - 61} L26 74 Q31 23 68 13Z`} fill="var(--seat-car-body)" stroke="var(--seat-car-trim)" strokeWidth="2" />
        <path d="M52 55Q118 35 184 55L191 86Q118 74 45 86Z" fill="var(--seat-car-glass)" />
        <path d={`M46 ${height - 66} Q118 ${height - 56} 190 ${height - 66} L183 ${height - 36} Q118 ${height - 22} 53 ${height - 36}Z`} fill="var(--seat-car-glass)" />
        <path d={`M35 101v${height - 182}M201 101v${height - 182}`} fill="none" stroke="var(--seat-car-trim)" strokeWidth="2" />
        <text x="118" y="31" textAnchor="middle" fontSize="9" letterSpacing="2" fill="var(--color-muted)">FRONT</text>
        <g transform="translate(40 104)" fill="var(--seat-driver)" color="var(--seat-driver)"><SeatShape /><circle cx="22" cy="14" r="11" fill="var(--seat-car-body)" stroke="var(--color-muted)" strokeWidth="2" /><path d="M12 14h20m-10 0v10" stroke="var(--color-muted)" strokeWidth="2" /></g>
        {!compact && <text x="62" y="161" textAnchor="middle" fontSize="8" fill="var(--color-muted)">DRIVER</text>}
        {compact && seats.map((seat) => {
          const state = seatState(seat.id, layout, selected, owned)
          return <g className={`seat-glyph seat-${state}`} key={seat.id} transform={`translate(${40 + seat.column * 56} ${104 + seat.row * 66})`}><title>Seat {seat.number}: {stateLabels[state]}</title><SeatShape /><text x="22" y="28" textAnchor="middle" fontSize="13" fill="var(--seat-label)" stroke="none">{state === 'owned' ? '✓' : state === 'booked' ? '×' : seat.number}</text></g>
        })}
      </svg>
      {!compact && seats.map((seat) => {
        const state = seatState(seat.id, layout, selected, owned)
        return <button key={seat.id} type="button" className={`car-seat seat-${state}`} style={{ left: `${(40 + seat.column * 56) / 236 * 100}%`, top: `${(104 + seat.row * 66) / height * 100}%` }} aria-label={`Seat ${seat.number}, ${stateLabels[state]}`} aria-pressed={state === 'selected'} disabled={disabled || state === 'booked' || state === 'owned' || !onToggle} onClick={() => onToggle(seat.id)}><svg viewBox="0 0 44 44" aria-hidden="true"><SeatShape /></svg><span>{state === 'owned' ? '✓' : state === 'booked' ? '×' : seat.number}</span></button>
      })}
    </div>
    {!compact && <ul className="seat-legend" aria-label="Seat colors">{Object.entries(stateLabels).map(([state, label]) => <li key={state}><span className={`seat-swatch seat-${state}`} />{label}</li>)}</ul>}
  </div>
}
