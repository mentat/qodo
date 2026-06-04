// Mapbox-based realistic globe for the Risk app.
//
// The center of the play screen is a Mapbox GL JS map projected as a globe,
// rendering the outdoors-v12 basemap (topographic / shaded relief) with
// 42 Risk territory polygons overlaid as a GeoJSON source. Fill color is
// driven by ownership; the selected territory gets a glowing pink border;
// army counts render as labels via a symbol layer.
//
// Public contract (consumed by GameScreen.tsx) is unchanged from v1:
//   <Globe onTerritoryClick={...} />

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import MapGL, {
  Layer, Source, type MapRef, type MapMouseEvent,
} from 'react-map-gl/mapbox';
import { Box, Center, Loader, Text } from '@mantine/core';
import { useRiskStore } from '../../store/riskStore';
import {
  loadTerritoryFeatures,
  type TerritoryFeatureCollection,
} from './territoryGeo';
import type { TerritoryID } from './board';
import type { GameState, Player } from '../../types/risk';

// Player → Mantine palette → hex map. Reads CSS variables at runtime so the
// dark/light theme flip is honored automatically.
function cssVar(name: string): string {
  if (typeof window === 'undefined') return '#888';
  const v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
  return v || '#888';
}
function playerColorHex(p: Player | undefined): string {
  if (!p) return cssVar('--mantine-color-gray-5');
  return cssVar(`--mantine-color-${p.color}-6`);
}

interface Props {
  onTerritoryClick: (t: TerritoryID) => void;
}

const MAPBOX_TOKEN = import.meta.env.VITE_MAPBOX_TOKEN as string | undefined;
const MAP_STYLE = 'mapbox://styles/mapbox/outdoors-v12';

export function Globe({ onTerritoryClick }: Props) {
  const game = useRiskStore((s) => s.game);
  const selectedFrom = useRiskStore((s) => s.selectedFrom);
  const mapRef = useRef<MapRef | null>(null);

  // 1) Lazy-load the 42 Risk-territory GeoJSON features once on mount.
  const [features, setFeatures] = useState<TerritoryFeatureCollection | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  useEffect(() => {
    let cancelled = false;
    loadTerritoryFeatures()
      .then((f) => { if (!cancelled) setFeatures(f); })
      .catch((e) => { if (!cancelled) setLoadError((e as Error).message); });
    return () => { cancelled = true; };
  }, []);

  // 2) Re-decorate features with current owner/armies/color whenever game changes.
  const decorated = useMemo<TerritoryFeatureCollection | null>(() => {
    if (!features || !game) return features;
    const playerById = new Map(game.players.map((p) => [p.id, p]));
    return {
      type: 'FeatureCollection',
      features: features.features.map((f) => {
        const ts = game.board[f.properties.id];
        const owner = ts ? playerById.get(ts.ownerId) : undefined;
        return {
          ...f,
          properties: {
            ...f.properties,
            ownerId: owner?.id ?? '',
            armies: ts?.armies ?? 0,
            color: playerColorHex(owner),
          },
        };
      }),
    };
  }, [features, game]);

  // 3) Click → translate Mapbox feature to TerritoryID and dispatch.
  const handleClick = useCallback(
    (e: MapMouseEvent) => {
      const feat = e.features?.[0];
      if (!feat) return;
      const id = (feat.properties?.id as TerritoryID | undefined);
      if (id) onTerritoryClick(id);
    },
    [onTerritoryClick],
  );

  // 4) Pointer cursor when hovering a territory.
  const handleMouseEnter = useCallback(() => {
    const map = mapRef.current?.getMap();
    if (map) map.getCanvas().style.cursor = 'pointer';
  }, []);
  const handleMouseLeave = useCallback(() => {
    const map = mapRef.current?.getMap();
    if (map) map.getCanvas().style.cursor = '';
  }, []);

  // 5) Auto-rotate the globe while idle (pauses on user drag/zoom).
  const userInteractingRef = useRef(false);
  useEffect(() => {
    let frame = 0;
    const tick = () => {
      const map = mapRef.current?.getMap();
      if (map && !userInteractingRef.current && map.isStyleLoaded()) {
        const c = map.getCenter();
        map.easeTo({ center: [c.lng + 0.06, c.lat], duration: 50, easing: (t) => t });
      }
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, []);

  if (!MAPBOX_TOKEN) {
    return (
      <Box style={{
        position: 'absolute', inset: 0, borderRadius: 16,
        background: 'var(--mantine-color-synthPurple-light)',
        display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24,
      }}>
        <Text size="sm" c="dimmed" ta="center">
          Mapbox token missing. Set <b>VITE_MAPBOX_TOKEN</b> in <code>frontend/.env</code>
          {' '}and reload. See <code>frontend/.env.example</code>.
        </Text>
      </Box>
    );
  }

  if (loadError) {
    return (
      <Center style={{ position: 'absolute', inset: 0 }}>
        <Text size="sm" c="neonPink.6">Failed to load globe data: {loadError}</Text>
      </Center>
    );
  }

  return (
    <Box style={{
      position: 'absolute', inset: 0, borderRadius: 16, overflow: 'hidden',
      boxShadow: 'inset 0 0 60px rgba(155, 93, 229, 0.25)',
    }}>
      <MapGL
        ref={mapRef}
        mapboxAccessToken={MAPBOX_TOKEN}
        mapStyle={MAP_STYLE}
        projection={{ name: 'globe' }}
        initialViewState={{ latitude: 20, longitude: 0, zoom: 1.4 }}
        onLoad={(e) => {
          // Synthwave-tinted atmosphere — Mapbox renders the globe's fog +
          // space backdrop based on these colors.
          e.target.setFog({
            color: 'rgb(26, 10, 61)',
            'high-color': 'rgb(155, 93, 229)',
            'horizon-blend': 0.04,
            'space-color': 'rgb(10, 0, 20)',
            'star-intensity': 0.6,
          });
        }}
        onMouseDown={() => { userInteractingRef.current = true; }}
        onMouseUp={() => { userInteractingRef.current = false; }}
        onTouchStart={() => { userInteractingRef.current = true; }}
        onTouchEnd={() => { userInteractingRef.current = false; }}
        interactiveLayerIds={['risk-fill']}
        onClick={handleClick}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
      >
        {decorated && (
          <Source id="risk" type="geojson" data={decorated} promoteId="id">
            <Layer
              id="risk-fill"
              type="fill"
              paint={{
                'fill-color': ['get', 'color'],
                'fill-opacity': [
                  'case',
                  ['==', ['get', 'id'], selectedFrom ?? ''], 0.78,
                  0.55,
                ],
              }}
            />
            <Layer
              id="risk-line"
              type="line"
              paint={{
                'line-color': 'rgba(255, 255, 255, 0.7)',
                'line-width': 1.2,
              }}
            />
            <Layer
              id="risk-selected-outline"
              type="line"
              filter={['==', ['get', 'id'], selectedFrom ?? '']}
              paint={{
                'line-color': '#ff00aa',
                'line-width': 3,
                'line-blur': 1.5,
              }}
            />
            <Layer
              id="risk-army-labels"
              type="symbol"
              layout={{
                'text-field': ['to-string', ['get', 'armies']],
                'text-size': 14,
                'text-font': ['DIN Pro Bold', 'Arial Unicode MS Bold'],
                'text-allow-overlap': true,
                'text-ignore-placement': true,
              }}
              paint={{
                'text-color': '#ffffff',
                'text-halo-color': '#000000',
                'text-halo-width': 1.4,
              }}
            />
          </Source>
        )}
      </MapGL>
      {!decorated && (
        <Center style={{ position: 'absolute', inset: 0, pointerEvents: 'none' }}>
          <Loader size="md" color="synthPurple" />
        </Center>
      )}
    </Box>
  );
}

// Re-export for any consumers that imported the helper from the v1 file.
// Used internally by GameScreen.tsx; safe no-op for anything else.
export type { GameState };
