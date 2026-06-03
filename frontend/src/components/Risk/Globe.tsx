import { useMemo, useRef } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import { OrbitControls, Html, Stars, Line } from '@react-three/drei';
import * as THREE from 'three';
import { TERRITORIES, latLonToVec3, type TerritoryID } from './board';
import { useRiskStore, humanPlayer } from '../../store/riskStore';
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

// ────── Globe ─────────────────────────────────────────────────────────────

interface Props {
  onTerritoryClick: (t: TerritoryID) => void;
}

export function Globe({ onTerritoryClick }: Props) {
  return (
    <div style={{
      position: 'absolute', inset: 0,
      borderRadius: 16, overflow: 'hidden',
      background: 'radial-gradient(circle at 30% 20%, #1a0033 0%, #0a0014 70%, #000 100%)',
      boxShadow: 'inset 0 0 60px rgba(155, 93, 229, 0.25)',
    }}>
      <Canvas
        camera={{ position: [0, 0, 3.6], fov: 45 }}
        gl={{ antialias: true, alpha: true }}
      >
        <ambientLight intensity={0.6} />
        <directionalLight position={[5, 3, 5]} intensity={1.0} />
        <directionalLight position={[-5, -2, -3]} intensity={0.3} color="#ff66cc" />
        <Stars radius={50} depth={20} count={1500} factor={3} fade speed={0.4} />

        <Scene onTerritoryClick={onTerritoryClick} />

        <OrbitControls
          enablePan={false}
          enableZoom
          enableDamping
          dampingFactor={0.08}
          autoRotate
          autoRotateSpeed={0.45}
          minDistance={2.4}
          maxDistance={6}
        />
      </Canvas>
    </div>
  );
}

function Scene({ onTerritoryClick }: { onTerritoryClick: (t: TerritoryID) => void }) {
  const game = useRiskStore((s) => s.game);
  const selectedFrom = useRiskStore((s) => s.selectedFrom);
  const human = humanPlayer(game);

  return (
    <group>
      <PlanetSphere />
      <Graticule />
      {game && TERRITORIES.map((t) => {
        const ts = game.board[t.id];
        const owner = game.players.find((p) => p.id === ts?.ownerId);
        const isSelected = selectedFrom === t.id;
        const isMine = owner?.id === human?.id;
        return (
          <Territory
            key={t.id}
            id={t.id}
            lat={t.lat}
            lon={t.lon}
            name={t.name}
            armies={ts?.armies ?? 0}
            color={playerColorHex(owner)}
            isSelected={isSelected}
            isMine={isMine}
            onClick={onTerritoryClick}
          />
        );
      })}
      {/* Adjacency arc from selected attacker to hovered/possible targets */}
      {game && selectedFrom && <SelectionArcs from={selectedFrom} game={game} />}
    </group>
  );
}

// ────── PlanetSphere ──────────────────────────────────────────────────────

function PlanetSphere() {
  const ref = useRef<THREE.Mesh>(null);
  // Slow extra rotation on top of OrbitControls.autoRotate — turns it into a
  // gently-spinning planet rather than a fixed camera orbit.
  useFrame((_, delta) => {
    if (ref.current) ref.current.rotation.y += delta * 0.02;
  });
  return (
    <mesh ref={ref}>
      <sphereGeometry args={[1, 64, 64]} />
      <meshStandardMaterial
        color="#1a0a3d"
        emissive="#1a0a3d"
        emissiveIntensity={0.25}
        roughness={0.7}
        metalness={0.15}
      />
    </mesh>
  );
}

// Graticule: faint synthwave-purple lat/lon grid drawn on the sphere.
function Graticule() {
  const segments = useMemo(() => {
    const out: [number, number, number][][] = [];
    // Latitudes
    for (let lat = -60; lat <= 60; lat += 30) {
      const ring: [number, number, number][] = [];
      for (let lon = -180; lon <= 180; lon += 5) {
        ring.push(latLonToVec3(lat, lon, 1.001));
      }
      out.push(ring);
    }
    // Longitudes
    for (let lon = -180; lon < 180; lon += 30) {
      const ring: [number, number, number][] = [];
      for (let lat = -90; lat <= 90; lat += 5) {
        ring.push(latLonToVec3(lat, lon, 1.001));
      }
      out.push(ring);
    }
    return out;
  }, []);
  return (
    <group>
      {segments.map((points, i) => (
        <Line key={i} points={points} color="#9b5de5" opacity={0.18} transparent lineWidth={1} />
      ))}
    </group>
  );
}

// ────── Territory marker ──────────────────────────────────────────────────

interface TerritoryProps {
  id: TerritoryID;
  lat: number;
  lon: number;
  name: string;
  armies: number;
  color: string;
  isSelected: boolean;
  isMine: boolean;
  onClick: (t: TerritoryID) => void;
}

function Territory({ id, lat, lon, armies, color, isSelected, isMine, onClick, name }: TerritoryProps) {
  const pos = useMemo<[number, number, number]>(() => latLonToVec3(lat, lon, 1.03), [lat, lon]);
  const ref = useRef<THREE.Mesh>(null);
  useFrame((_, dt) => {
    if (ref.current && isSelected) {
      ref.current.rotation.y += dt * 1.6;
    }
  });

  return (
    <group position={pos}>
      <mesh
        ref={ref}
        onPointerDown={(e) => { e.stopPropagation(); onClick(id); }}
        onPointerOver={(e) => { e.stopPropagation(); document.body.style.cursor = 'pointer'; }}
        onPointerOut={() => { document.body.style.cursor = ''; }}
      >
        <sphereGeometry args={[isSelected ? 0.045 : 0.035, 16, 16]} />
        <meshStandardMaterial
          color={color}
          emissive={color}
          emissiveIntensity={isSelected ? 1.2 : isMine ? 0.7 : 0.45}
          roughness={0.3}
          metalness={0.6}
        />
      </mesh>
      <Html
        center
        position={[0, 0.06, 0]}
        style={{
          pointerEvents: 'none',
          fontSize: 11,
          fontWeight: 800,
          color: 'white',
          textShadow: '0 0 6px rgba(0,0,0,0.95), 0 0 2px rgba(0,0,0,1)',
          whiteSpace: 'nowrap',
          userSelect: 'none',
        }}
      >
        <div style={{
          padding: '2px 6px',
          borderRadius: 6,
          background: `${color}d8`,
          border: isSelected ? '1.5px solid white' : '1px solid rgba(255,255,255,0.25)',
          display: 'flex', alignItems: 'center', gap: 4,
        }}>
          <span style={{ fontSize: 10, opacity: 0.85 }}>{abbrev(name)}</span>
          <span style={{ fontSize: 12 }}>{armies}</span>
        </div>
      </Html>
    </group>
  );
}

// abbrev shortens a territory name for the in-globe label — long ones get
// initials, short ones stay full.
function abbrev(name: string): string {
  if (name.length <= 11) return name;
  return name
    .split(' ')
    .map((w) => (w.length <= 3 ? w : w[0].toUpperCase()))
    .join(' ');
}

// SelectionArcs: glowing great-circle arcs from the selected attacker to each
// adjacent enemy territory, drawn as 3D lines on the sphere.
function SelectionArcs({ from, game }: { from: TerritoryID; game: GameState }) {
  const fromDef = TERRITORIES.find((t) => t.id === from);
  if (!fromDef) return null;
  const fromVec = latLonToVec3(fromDef.lat, fromDef.lon, 1.03);
  return (
    <group>
      {fromDef.adjacent.map((adj) => {
        const adjDef = TERRITORIES.find((t) => t.id === adj);
        if (!adjDef) return null;
        const toVec = latLonToVec3(adjDef.lat, adjDef.lon, 1.03);
        const target = game.board[adj];
        const targetOwner = game.players.find((p) => p.id === target?.ownerId);
        const isEnemy = targetOwner && targetOwner.kind !== 'human' && targetOwner.id !== game.players.find((p) => p.kind === 'human')?.id;
        const color = isEnemy ? '#ff00aa' : '#9b5de5';
        return <Arc key={adj} from={fromVec} to={toVec} color={color} />;
      })}
    </group>
  );
}

// Arc: a curved line traveling along the sphere's surface between two points,
// approximated by sampling slerp on the unit sphere.
function Arc({ from, to, color }: {
  from: [number, number, number]; to: [number, number, number]; color: string;
}) {
  const points = useMemo(() => {
    const v1 = new THREE.Vector3(...from).normalize();
    const v2 = new THREE.Vector3(...to).normalize();
    const omega = Math.acos(Math.min(1, Math.max(-1, v1.dot(v2))));
    const sinO = Math.sin(omega);
    const N = 24;
    const out: [number, number, number][] = [];
    for (let i = 0; i <= N; i++) {
      const t = i / N;
      const a = sinO === 0 ? 1 - t : Math.sin((1 - t) * omega) / sinO;
      const b = sinO === 0 ? t : Math.sin(t * omega) / sinO;
      const v = v1.clone().multiplyScalar(a).add(v2.clone().multiplyScalar(b));
      // Lift the midpoint slightly off the surface so the arc reads as a "hop".
      const lift = 1 + 0.05 * Math.sin(t * Math.PI);
      v.multiplyScalar(lift);
      out.push([v.x, v.y, v.z]);
    }
    return out;
  }, [from, to]);
  return <Line points={points} color={color} lineWidth={2} transparent opacity={0.85} />;
}
