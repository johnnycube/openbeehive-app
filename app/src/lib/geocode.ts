// Forward + reverse geocoding via OSM Nominatim. Usage stays within the
// public instance's fair-use policy: requests are user-triggered (typing an
// address / dropping a pin), debounced by the caller, and identified by the
// browser's Referer. If the service is unreachable the helpers resolve to
// null and the UI falls back to raw GPS coordinates.
export type GeoHit = { lat: number; lng: number; label: string };

const BASE = 'https://nominatim.openstreetmap.org';

async function ask(path: string): Promise<any | null> {
  try {
    const r = await fetch(`${BASE}${path}`, {
      headers: { Accept: 'application/json', 'Accept-Language': navigator.language ?? 'en' },
    });
    if (!r.ok) return null;
    return await r.json();
  } catch {
    return null;
  }
}

/** Best match for a free-form address, or null. */
export async function geocode(query: string): Promise<GeoHit | null> {
  const q = query.trim();
  if (q.length < 3) return null;
  const j = await ask(`/search?format=jsonv2&limit=1&q=${encodeURIComponent(q)}`);
  const hit = Array.isArray(j) ? j[0] : null;
  if (!hit) return null;
  return { lat: parseFloat(hit.lat), lng: parseFloat(hit.lon), label: hit.display_name ?? q };
}

/** Human-readable address for a position, or null (caller shows coordinates). */
export async function reverseGeocode(lat: number, lng: number): Promise<string | null> {
  const j = await ask(`/reverse?format=jsonv2&lat=${lat}&lon=${lng}`);
  return j?.display_name ?? null;
}

export const fmtCoords = (lat: number, lng: number): string =>
  `${lat.toFixed(5)}, ${lng.toFixed(5)}`;
