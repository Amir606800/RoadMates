import { dateAfter } from '../utils/format.js'

export const mockReviews = [
  { id: 'r1', target: 'client', author: 'Elvin Aliyev', rating: 5, date: dateAfter(-2), route: 'Baku → Ganja', comment: 'Leyla was right on time and lovely to travel with. A great mate for the road.', tags: ['Punctual', 'Polite'] },
  { id: 'r2', target: 'client', author: 'Nigar Hasanli', rating: 5, date: dateAfter(-8), route: 'Ganja → Baku', comment: 'Clear communication and a very pleasant journey. Welcome back anytime!', tags: ['Polite', 'Reliable'] },
  { id: 'r3', target: 'client', author: 'Murad Karimov', rating: 4, date: dateAfter(-14), route: 'Baku → Gabala', comment: 'A friendly passenger. Let me know in advance when plans changed.', tags: ['Good communication'] },
  { id: 'r4', target: 'driver', author: 'Aysel Rahimova', rating: 5, date: dateAfter(-3), route: 'Ganja → Baku', comment: 'A comfortable car and a calm, careful driver. Would happily book again.', tags: ['Safe driving', 'Comfortable'] },
  { id: 'r5', target: 'driver', author: 'Rashad Mammadov', rating: 5, date: dateAfter(-9), route: 'Baku → Gabala', comment: 'On time, friendly, and a beautiful drive through the mountains.', tags: ['On time'] },
  { id: 'r6', target: 'driver', author: 'Fidan Aliyeva', rating: 4, date: dateAfter(-16), route: 'Baku → Ganja', comment: 'Everything went smoothly. Plenty of room for my bag too.', tags: ['Comfortable'] },
]

export const mockCompletedTrips = [
  { id: 'trip-c1', role: 'client', person: 'Elvin Aliyev', from: 'Baku', to: 'Ganja', date: dateAfter(-2) },
  { id: 'trip-c2', role: 'client', person: 'Nigar Hasanli', from: 'Ganja', to: 'Baku', date: dateAfter(-8) },
  { id: 'trip-d1', role: 'driver', person: 'Aysel Rahimova', from: 'Ganja', to: 'Baku', date: dateAfter(-3) },
]
