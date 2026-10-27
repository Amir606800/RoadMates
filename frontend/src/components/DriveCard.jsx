import { Link } from 'react-router-dom'
import Avatar from './Avatar.jsx'
import Icon from './Icon.jsx'
import { formatDate, money } from '../utils/format.js'
import { driverName, rideDate, rideTime } from '../api/ridePayload.js'

export default function DriveCard({ drive }) {
  const name = driverName(drive)
  return <Link to={`/drives/${drive.id}`} className="card drive-card" aria-label={`${drive.from_district.name_az} to ${drive.to_district.name_az}, ${rideTime(drive.departure_at)}, ${name}, ${money(drive.price_azn)} per seat`}>
    <div className="row between"><div className="row"><Avatar name={name} /><div><h3>{name}</h3><span className="text-sm muted">{formatDate(rideDate(drive.departure_at))} · Azerbaijan time</span></div></div><span className="verified-mark" aria-label="Verified driver"><Icon name="shield" size={19} /></span></div>
    <div className="ride-route"><div><strong>{drive.from_district.name_az}</strong><span>{rideTime(drive.departure_at)} departure</span></div><Icon name="arrow" size={18} /><div><strong>{drive.to_district.name_az}</strong><span>Destination</span></div></div>
    {(drive.from_district.requires_permit || drive.to_district.requires_permit) && <p className="field-hint permit-hint">Permit may be required for this route</p>}
    <div className="drive-footer"><span className="seats-label"><Icon name="users" size={15} /> {drive.seats_available} of {drive.seats_total} seats available</span><div className="price"><strong>{money(drive.price_azn)}</strong><span>per seat</span></div></div>
  </Link>
}
