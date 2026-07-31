<script lang="ts">
  import 'leaflet/dist/leaflet.css';
  import { onMount, onDestroy } from 'svelte';
  import { _ } from 'svelte-i18n';
  import { geocode, reverseGeocode, fmtCoords } from '$lib/geocode';

  // Two-way bound: the parent's address input and the pin stay in sync.
  // Typing an address geocodes it and moves the pin; moving the pin (drag or
  // map click) reverse-geocodes and rewrites the address — or falls back to
  // plain GPS coordinates when no address is known for the spot.
  let {
    lat = $bindable(0),
    lng = $bindable(0),
    address = $bindable(''),
  }: { lat?: number; lng?: number; address?: string } = $props();

  let el: HTMLDivElement;
  let map: any = null;
  let L: any = null;
  let marker: any = null;
  let busy = $state(false);

  // The address value we last wrote ourselves (reverse geocode). Seeing it
  // again in the effect means "not user input" — do not geocode it back.
  let selfWritten = '';
  let debounceTimer: ReturnType<typeof setTimeout> | undefined;

  function pin(Lib: any) {
    return Lib.divIcon({
      className: 'obh-pin', html: '<span></span>', iconSize: [26, 26], iconAnchor: [13, 26]
    });
  }

  function placeMarker(a: number, b: number, pan = true) {
    if (!map || !L) return;
    if (!marker) {
      marker = L.marker([a, b], { icon: pin(L), draggable: true }).addTo(map);
      marker.on('dragend', () => {
        const p = marker.getLatLng();
        pinMoved(p.lat, p.lng);
      });
    } else {
      marker.setLatLng([a, b]);
    }
    if (pan) map.setView([a, b], Math.max(map.getZoom(), 14));
  }

  async function pinMoved(a: number, b: number) {
    lat = +a.toFixed(6);
    lng = +b.toFixed(6);
    busy = true;
    const label = (await reverseGeocode(lat, lng)) ?? fmtCoords(lat, lng);
    selfWritten = label;
    address = label;
    busy = false;
  }

  // Coordinates changed from outside (numeric inputs, geolocation button) ->
  // move the pin. A no-op when the pin itself caused the change.
  $effect(() => {
    const a = lat, b = lng;
    if (!map || (!a && !b)) return;
    const cur = marker?.getLatLng();
    if (!cur || Math.abs(cur.lat - a) > 1e-6 || Math.abs(cur.lng - b) > 1e-6) {
      placeMarker(a, b);
    }
  });

  // Address typed by the user -> geocode (debounced) -> move the pin.
  $effect(() => {
    const a = address;
    if (!map || a === selfWritten || a.trim().length < 3) return;
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(async () => {
      busy = true;
      const hit = await geocode(a);
      busy = false;
      if (!hit) return;
      lat = +hit.lat.toFixed(6);
      lng = +hit.lng.toFixed(6);
      placeMarker(hit.lat, hit.lng);
    }, 700);
  });

  onMount(async () => {
    L = (await import('leaflet')).default;
    const hasPos = !!(lat || lng);
    map = L.map(el, { zoomControl: true, attributionControl: true })
      .setView(hasPos ? [lat, lng] : [50.5, 9.5], hasPos ? 14 : 5);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      maxZoom: 19, attribution: '© OpenStreetMap'
    }).addTo(map);
    if (hasPos) placeMarker(lat, lng, false);
    map.on('click', (e: any) => {
      placeMarker(e.latlng.lat, e.latlng.lng, false);
      pinMoved(e.latlng.lat, e.latlng.lng);
    });
    setTimeout(() => map?.invalidateSize(), 60);
  });

  onDestroy(() => {
    clearTimeout(debounceTimer);
    map?.remove();
    map = null;
  });
</script>

<div class="picker">
  <div class="map" bind:this={el}></div>
  <p class="hint">
    {#if busy}⌖ …{:else}{$_('apiaries.map_hint')}{/if}
  </p>
</div>

<style>
  .map { height: 240px; width: 100%; border-radius: 12px; overflow: hidden;
    border: 1px solid var(--line); z-index: 0; }
  .hint { color: var(--ink-soft); font-size: .8rem; margin: 6px 2px 0; }
  :global(.obh-pin span) {
    display: block; width: 20px; height: 20px; border-radius: 50% 50% 50% 0;
    background: var(--honey, #c77f22); border: 2px solid #fffdf7; transform: rotate(-45deg);
    box-shadow: 0 2px 5px rgba(0,0,0,.3);
  }
</style>
