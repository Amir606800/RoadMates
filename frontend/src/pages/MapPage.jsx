import { Link, useLocation } from 'react-router-dom'
import { useApp } from '../context/AppContext.jsx'
import Icon from '../components/Icon.jsx'

export default function MapPage() {
  const { search } = useApp()
  const { state } = useLocation()
  const from = state?.from || search.from?.name_az
  const to = state?.to || search.to?.name_az
  return <>
    <section className="page-intro"><span className="eyebrow">SEE WHERE THE ROAD GOES</span><h1>A shared path.<br />A new perspective.</h1><p>Your journey, from the first hello to arrival.</p></section>
    {/* TODO: replace this schematic with an open-source map such as Leaflet.
        Resolve selected city coordinates in the API layer, then render real
        tiles and route markers. This preview is not a geographic map. */}
    <section className="map-board" aria-label={`Illustrative route preview from ${from || 'departure'} to ${to || 'destination'}`}>
      <svg className="map-drawing" viewBox="0 0 350 400" preserveAspectRatio="xMidYMid slice" aria-hidden="true"><rect width="350" height="400" fill="#eaf0e4" /><path d="M260-10c-20 70-10 110 25 150s-10 70-15 140 20 90 50 140" stroke="#d1e4e4" strokeWidth="44" fill="none" /><g fill="#dce8d5"><path d="m10 20 90-10 20 65-90 20Z" /><path d="m25 200 80-5 20 65-85 30Z" /><path d="m90 310 90-15 30 90-90 15Z" /><path d="m170 55 40-10 15 35-45 10Z" /></g><g stroke="#fff" strokeWidth="13" fill="none"><path d="m-20 140 400-90M30-20l130 450M160-10l90 420M-10 300l400-110M-10 380l400-110" /></g><g stroke="#d8dfd1" strokeWidth="1.5" fill="none"><path d="m-20 140 400-90M30-20l130 450M160-10l90 420M-10 300l400-110M-10 380l400-110" /></g><path d="M94 113 129 248 235 218 256 316" stroke="#245b45" strokeWidth="5" fill="none" strokeLinecap="round" strokeLinejoin="round" /><path d="M94 113 129 248 235 218 256 316" stroke="#fff" strokeWidth="1.5" fill="none" strokeDasharray="4 7" /><circle cx="94" cy="113" r="7" fill="#245b45" stroke="white" strokeWidth="3" /><circle cx="256" cy="316" r="7" fill="#245b45" stroke="white" strokeWidth="3" /></svg>
      <span className="map-preview-label"><Icon name="map" size={13} />Illustrative preview</span>
      {from && <span className="map-pin from"><span className="map-pin-letter">A</span><span>{from}</span></span>}
      {to && <span className="map-pin to"><span className="map-pin-letter">B</span><span>{to}</span></span>}
      <span className="map-caption">A little closer, together.</span>
    </section>
    <section className="card map-route-card stack"><div className="row"><span className="settings-icon"><Icon name="road" size={22} /></span><div><h2>{from || 'Choose departure'} → {to || 'Choose destination'}</h2><p className="text-sm muted">{state?.from ? 'Selected drive route' : 'Your selected search route'}</p></div></div><div className="notice"><Icon name="info" size={18} /><p>A real map is coming later. This is a route placeholder with illustrative pins.</p></div><Link className="button full-width" to="/drives">Find your drive<Icon name="arrow" size={18} /></Link></section>
  </>
}
