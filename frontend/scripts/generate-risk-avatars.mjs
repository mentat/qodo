import { mkdir, writeFile } from 'node:fs/promises';
import { DotLottie } from '@dotlottie/dotlottie-js';

const SIZE = 256;
const FPS = 30;
const OUT_DIR = new URL('../public/risk-avatars/', import.meta.url);

const STATES = {
  idle: { frames: 90, loop: true },
  hover: { frames: 48, loop: true },
  thinking: { frames: 60, loop: true },
  nod: { frames: 36, loop: false },
  shake: { frames: 32, loop: false },
  startled: { frames: 36, loop: false },
};

const GENERALS = [
  {
    id: 'maxine-voltage',
    name: 'Maxine Voltage',
    accent: '#ff00aa',
    secondary: '#00e5ff',
    skin: '#f4a7c8',
    uniform: '#2d1245',
    hair: '#ff66cc',
    eye: '#39ff14',
    symbol: 'bolt',
  },
  {
    id: 'general-static',
    name: 'General Static',
    accent: '#4d9cff',
    secondary: '#ff00aa',
    skin: '#b8d8ff',
    uniform: '#111f43',
    hair: '#1a80ff',
    eye: '#00e5ff',
    symbol: 'screen',
    visor: true,
  },
  {
    id: 'vice-admiral-vector',
    name: 'Vice-Admiral Vector',
    accent: '#00cc66',
    secondary: '#00e5ff',
    skin: '#bfe8d2',
    uniform: '#123a34',
    hair: '#11c485',
    eye: '#fff14d',
    symbol: 'anchor',
    cap: true,
  },
  {
    id: 'commodore-cassette',
    name: 'Commodore Cassette',
    accent: '#ffe600',
    secondary: '#ff66cc',
    skin: '#f0c68f',
    uniform: '#3f2b12',
    hair: '#ffb703',
    eye: '#00e5ff',
    symbol: 'cassette',
  },
  {
    id: 'captain-coral',
    name: 'Captain Coral',
    accent: '#9c4dff',
    secondary: '#33ffcc',
    skin: '#f0a6b8',
    uniform: '#2c1748',
    hair: '#ff7aa8',
    eye: '#b3ffb3',
    symbol: 'shell',
  },
  {
    id: 'field-marshal-neon',
    name: 'Field Marshal Neon',
    accent: '#ff00aa',
    secondary: '#ffe600',
    skin: '#ffc0d8',
    uniform: '#39123f',
    hair: '#d4b3ff',
    eye: '#00e5ff',
    symbol: 'medal',
    cap: true,
  },
  {
    id: 'colonel-chrome',
    name: 'Colonel Chrome',
    accent: '#80b8ff',
    secondary: '#b8c7df',
    skin: '#c9d7e8',
    uniform: '#17253f',
    hair: '#d7e4f5',
    eye: '#39ff14',
    symbol: 'trident',
    visor: true,
  },
  {
    id: 'lieutenant-laser',
    name: 'Lt. Laser',
    accent: '#ffe600',
    secondary: '#ff00aa',
    skin: '#f6b78a',
    uniform: '#3a1a2e',
    hair: '#ffed1a',
    eye: '#ff2e97',
    symbol: 'target',
  },
];

await mkdir(OUT_DIR, { recursive: true });

for (const general of GENERALS) {
  const dotLottie = new DotLottie({ generator: 'qodo-demo-risk-avatar-generator@1.0.0' });

  for (const [state, config] of Object.entries(STATES)) {
    dotLottie.addAnimation({
      id: state,
      data: createAnimation(general, state, config.frames),
    });
  }

  dotLottie.addStateMachine({
    id: 'risk-avatar',
    name: 'Risk Avatar',
    data: createStateMachine(),
  });

  await dotLottie.build();
  const arrayBuffer = await dotLottie.toArrayBuffer();
  await writeFile(new URL(`${general.id}.lottie`, OUT_DIR), Buffer.from(arrayBuffer));
}

console.log(`Generated ${GENERALS.length} Risk avatars in ${OUT_DIR.pathname}`);

function createAnimation(general, state, frames) {
  const layers = [];
  let ind = 1;
  const motion = motionFor(state, frames);

  layers.push(ringLayer(general, ind++, frames, state));
  layers.push(torsoLayer(general, ind++, frames, motion));
  layers.push(neckLayer(general, ind++, frames, motion));
  layers.push(headLayer(general, ind++, frames, motion));
  layers.push(hairLayer(general, ind++, frames, motion));
  if (general.visor) layers.push(visorLayer(general, ind++, frames, motion));
  layers.push(eyeLayer(general, ind++, frames, motion, 'left', state));
  layers.push(eyeLayer(general, ind++, frames, motion, 'right', state));
  layers.push(mouthLayer(general, ind++, frames, motion, state));
  layers.push(symbolLayer(general, ind++, frames, motion));

  return {
    v: '5.12.2',
    fr: FPS,
    ip: 0,
    op: frames,
    w: SIZE,
    h: SIZE,
    nm: `${general.name} ${state}`,
    ddd: 0,
    assets: [],
    layers: layers.reverse(),
    markers: [],
  };
}

function createStateMachine() {
  const bool = (inputName, compareTo) => ({
    type: 'Boolean',
    inputName,
    conditionType: 'Equal',
    compareTo,
  });
  const mood = (compareTo) => ({
    type: 'String',
    inputName: 'eventMood',
    conditionType: 'Equal',
    compareTo,
  });
  const to = (toState, guards = []) => ({ type: 'Transition', toState, guards });
  const reactionTransitions = [
    to('nod', [mood('nod')]),
    to('shake', [mood('shake')]),
    to('startled', [mood('startled')]),
  ];
  const returnTransitions = [
    to('thinking', [mood('none'), bool('isThinking', true)]),
    to('thinking', [mood('none'), bool('isActive', true)]),
    to('hover', [mood('none'), bool('isHovered', true)]),
    to('idle', [mood('none')]),
  ];

  return {
    initial: 'idle',
    states: [
      {
        name: 'idle',
        type: 'PlaybackState',
        animation: 'idle',
        loop: true,
        autoplay: true,
        transitions: [
          ...reactionTransitions,
          to('thinking', [bool('isThinking', true)]),
          to('thinking', [bool('isActive', true)]),
          to('hover', [bool('isHovered', true)]),
        ],
      },
      {
        name: 'hover',
        type: 'PlaybackState',
        animation: 'hover',
        loop: true,
        autoplay: true,
        transitions: [
          ...reactionTransitions,
          to('thinking', [bool('isThinking', true)]),
          to('thinking', [bool('isActive', true)]),
          to('idle', [bool('isHovered', false)]),
        ],
      },
      {
        name: 'thinking',
        type: 'PlaybackState',
        animation: 'thinking',
        loop: true,
        autoplay: true,
        speed: 1.15,
        transitions: [
          ...reactionTransitions,
          to('hover', [bool('isThinking', false), bool('isActive', false), bool('isHovered', true)]),
          to('idle', [bool('isThinking', false), bool('isActive', false)]),
        ],
      },
      reactionState('nod', returnTransitions),
      reactionState('shake', returnTransitions),
      reactionState('startled', returnTransitions),
    ],
    interactions: ['nod', 'shake', 'startled'].map((stateName) => ({
      type: 'OnComplete',
      stateName,
      actions: [{ type: 'SetString', inputName: 'eventMood', value: 'none' }],
    })),
    inputs: [
      { type: 'Boolean', name: 'isActive', value: false },
      { type: 'Boolean', name: 'isThinking', value: false },
      { type: 'Boolean', name: 'isHovered', value: false },
      { type: 'String', name: 'eventMood', value: 'none' },
    ],
  };
}

function reactionState(name, transitions) {
  return {
    name,
    type: 'PlaybackState',
    animation: name,
    loop: false,
    autoplay: true,
    transitions,
  };
}

function motionFor(state, frames) {
  switch (state) {
    case 'hover':
      return [
        { t: 0, x: 0, y: -2, r: 0, s: 103 },
        { t: frames / 2, x: 0, y: -5, r: 1, s: 104 },
        { t: frames, x: 0, y: -2, r: 0, s: 103 },
      ];
    case 'thinking':
      return [
        { t: 0, x: -2, y: -1, r: -2, s: 101 },
        { t: frames * 0.25, x: 3, y: -4, r: 2.5, s: 101 },
        { t: frames * 0.5, x: -3, y: -2, r: -1.5, s: 101 },
        { t: frames * 0.75, x: 2, y: -5, r: 2, s: 101 },
        { t: frames, x: -2, y: -1, r: -2, s: 101 },
      ];
    case 'nod':
      return [
        { t: 0, x: 0, y: -1, r: 0, s: 101 },
        { t: frames * 0.25, x: 0, y: 7, r: 4, s: 102 },
        { t: frames * 0.55, x: 0, y: -5, r: -3, s: 101 },
        { t: frames, x: 0, y: 0, r: 0, s: 100 },
      ];
    case 'shake':
      return [
        { t: 0, x: 0, y: 0, r: 0, s: 100 },
        { t: frames * 0.2, x: -8, y: -1, r: -7, s: 101 },
        { t: frames * 0.4, x: 8, y: -1, r: 7, s: 101 },
        { t: frames * 0.6, x: -6, y: 0, r: -5, s: 100 },
        { t: frames * 0.8, x: 5, y: 0, r: 4, s: 100 },
        { t: frames, x: 0, y: 0, r: 0, s: 100 },
      ];
    case 'startled':
      return [
        { t: 0, x: 0, y: 0, r: 0, s: 100 },
        { t: frames * 0.18, x: 0, y: -8, r: 0, s: 108 },
        { t: frames * 0.55, x: 0, y: 2, r: 0, s: 101 },
        { t: frames, x: 0, y: 0, r: 0, s: 100 },
      ];
    default:
      return [
        { t: 0, x: 0, y: 0, r: 0, s: 100 },
        { t: frames / 2, x: 0, y: -4, r: 1, s: 101 },
        { t: frames, x: 0, y: 0, r: 0, s: 100 },
      ];
  }
}

function ringLayer(general, ind, frames, state) {
  const opacity = state === 'thinking' ? pulseProp(35, 80, frames) : pulseProp(24, 48, frames);
  return shapeLayer({
    name: 'neon command ring',
    ind,
    frames,
    position: prop([128, 128, 0]),
    opacity,
    shapes: [
      group([
        ellipse([0, 0], [210, 210]),
        stroke(general.accent, 5, 60),
      ], 'outer ring'),
      group([
        ellipse([0, 0], [174, 174]),
        stroke(general.secondary, 2.5, 45),
      ], 'inner ring'),
      group([
        path([[-80, 54], [80, 54]], false),
        stroke(general.accent, 2, 28),
      ], 'horizon line'),
    ],
  });
}

function torsoLayer(general, ind, frames, motion) {
  return motionLayer({
    name: 'uniform torso',
    ind,
    frames,
    base: [128, 205],
    motion,
    scaleFactor: 0.55,
    shapes: [
      group([
        rect([0, 0], [128, 58], 18),
        fill(general.uniform, 100),
        stroke(general.accent, 4, 80),
      ], 'torso'),
      group([
        path([[-54, -5], [54, -5]], false),
        stroke(general.secondary, 3, 80),
      ], 'rank stripe'),
    ],
  });
}

function neckLayer(general, ind, frames, motion) {
  return motionLayer({
    name: 'neck',
    ind,
    frames,
    base: [128, 170],
    motion,
    scaleFactor: 0.7,
    shapes: [
      group([
        rect([0, 0], [38, 38], 8),
        fill(general.skin, 100),
        stroke(general.accent, 2, 50),
      ], 'neck'),
    ],
  });
}

function headLayer(general, ind, frames, motion) {
  return motionLayer({
    name: 'head',
    ind,
    frames,
    base: [128, 116],
    motion,
    shapes: [
      group([
        ellipse([0, 0], [100, 112]),
        fill(general.skin, 100),
        stroke(general.accent, 4, 90),
      ], 'face'),
      group([
        ellipse([-28, 12], [14, 9]),
        fill(general.secondary, 30),
      ], 'left cheek'),
      group([
        ellipse([28, 12], [14, 9]),
        fill(general.secondary, 30),
      ], 'right cheek'),
    ],
  });
}

function hairLayer(general, ind, frames, motion) {
  const capTop = general.cap
    ? [[-46, -18], [-34, -54], [34, -54], [46, -18]]
    : [[-50, -20], [-28, -55], [22, -59], [50, -22], [40, 1], [-42, 1]];
  return motionLayer({
    name: general.cap ? 'command cap' : 'neon hair',
    ind,
    frames,
    base: [128, 116],
    motion,
    shapes: [
      group([
        path(capTop, true),
        fill(general.hair, 100),
        stroke(general.accent, 3, 88),
      ], 'top silhouette'),
      group([
        rect([0, -17], [92, 18], 8),
        fill(general.cap ? general.uniform : general.hair, 100),
        stroke(general.secondary, 2, 72),
      ], 'visor brim'),
    ],
  });
}

function visorLayer(general, ind, frames, motion) {
  return motionLayer({
    name: 'visor',
    ind,
    frames,
    base: [128, 110],
    motion,
    shapes: [
      group([
        rect([0, 0], [74, 24], 9),
        fill('#050714', 92),
        stroke(general.secondary, 2.5, 90),
      ], 'visor glass'),
      group([
        path([[-25, 0], [25, 0]], false),
        stroke(general.eye, 3, 90),
      ], 'visor scanline'),
    ],
  });
}

function eyeLayer(general, ind, frames, motion, side, state) {
  const baseX = side === 'left' ? 109 : 147;
  const thinkingLook = state === 'thinking' ? (side === 'left' ? -3 : 3) : 0;
  const eyeScale = eyeScaleProp(state, frames);
  const baseSize = state === 'startled' ? [17, 17] : [13, 13];
  return shapeLayer({
    name: `${side} eye`,
    ind,
    frames,
    position: motionPosition([baseX + thinkingLook, 107], motion),
    rotation: motionRotation(motion),
    scale: eyeScale,
    shapes: [
      group([
        ellipse([0, 0], baseSize),
        fill(general.eye, 100),
        stroke(general.secondary, 1.5, 75),
      ], 'eye glow'),
    ],
  });
}

function mouthLayer(general, ind, frames, motion, state) {
  const shapes = state === 'startled'
    ? [
        group([
          ellipse([0, 0], [18, 22]),
          fill('#060611', 100),
          stroke(general.secondary, 3, 95),
        ], 'startled mouth'),
      ]
    : [
        group([
          mouthPath(state),
          stroke(general.secondary, 4, 95),
        ], 'mouth line'),
      ];

  return motionLayer({
    name: 'mouth',
    ind,
    frames,
    base: [128, 138],
    motion,
    shapes,
  });
}

function symbolLayer(general, ind, frames, motion) {
  return motionLayer({
    name: `${general.symbol} insignia`,
    ind,
    frames,
    base: [128, 203],
    motion,
    scaleFactor: 0.55,
    shapes: symbolShapes(general),
  });
}

function symbolShapes(general) {
  const a = general.accent;
  const b = general.secondary;
  switch (general.symbol) {
    case 'bolt':
      return [group([path([[-8, -18], [10, -18], [1, -2], [14, -2], [-5, 20], [0, 4], [-13, 4]], true), fill(a, 100)], 'bolt')];
    case 'screen':
      return [
        group([rect([0, 0], [36, 25], 4), stroke(a, 3, 100)], 'screen'),
        group([path([[-12, 0], [-3, 0], [4, -5], [12, -5]], false), stroke(b, 2, 100)], 'static'),
      ];
    case 'anchor':
      return [
        group([path([[0, -18], [0, 14]], false), stroke(a, 3, 100)], 'anchor stem'),
        group([path([[-14, -3], [14, -3]], false), stroke(a, 3, 100)], 'anchor crossbar'),
        group([path([[-18, 8], [-9, 18], [0, 14], [9, 18], [18, 8]], false), stroke(b, 3, 100)], 'anchor fluke'),
      ];
    case 'cassette':
      return [
        group([rect([0, 0], [42, 26], 4), stroke(a, 3, 100)], 'cassette body'),
        group([ellipse([-11, 1], [9, 9]), fill(b, 100)], 'left reel'),
        group([ellipse([11, 1], [9, 9]), fill(b, 100)], 'right reel'),
      ];
    case 'shell':
      return [
        group([path([[0, -17], [-19, 15], [19, 15]], true), fill(a, 80), stroke(b, 2, 100)], 'shell fan'),
        group([path([[0, -14], [0, 15]], false), stroke(b, 2, 100)], 'shell rib'),
        group([path([[-9, -7], [-15, 15]], false), stroke(b, 2, 100)], 'shell rib left'),
        group([path([[9, -7], [15, 15]], false), stroke(b, 2, 100)], 'shell rib right'),
      ];
    case 'medal':
      return [
        group([ellipse([0, 4], [25, 25]), fill(a, 100), stroke(b, 2, 100)], 'medal'),
        group([path([[-10, -18], [0, -5], [10, -18]], false), stroke(b, 3, 100)], 'ribbon'),
      ];
    case 'trident':
      return [
        group([path([[0, -18], [0, 19]], false), stroke(a, 3, 100)], 'center tine'),
        group([path([[-15, -13], [-15, 2], [0, 8], [15, 2], [15, -13]], false), stroke(b, 3, 100)], 'side tines'),
      ];
    case 'target':
      return [
        group([ellipse([0, 0], [36, 36]), stroke(a, 3, 100)], 'target outer'),
        group([ellipse([0, 0], [18, 18]), stroke(b, 2, 100)], 'target inner'),
        group([path([[-21, 0], [21, 0]], false), stroke(b, 1.5, 80)], 'target horizontal'),
        group([path([[0, -21], [0, 21]], false), stroke(b, 1.5, 80)], 'target vertical'),
      ];
    default:
      return [group([ellipse([0, 0], [24, 24]), fill(a, 100)], 'dot')];
  }
}

function mouthPath(state) {
  if (state === 'shake') {
    return bezierPath([[-20, 3], [0, -6], [20, 3]], [[0, 0], [-10, 0], [0, 0]], [[0, 0], [10, 0], [0, 0]], false);
  }
  if (state === 'idle') {
    return path([[-16, 0], [16, 0]], false);
  }
  return bezierPath([[-22, -2], [0, 10], [22, -2]], [[0, 0], [-10, 0], [0, 0]], [[0, 0], [10, 0], [0, 0]], false);
}

function motionLayer({ name, ind, frames, base, motion, shapes, scaleFactor = 1 }) {
  return shapeLayer({
    name,
    ind,
    frames,
    position: motionPosition(base, motion, scaleFactor),
    rotation: motionRotation(motion, scaleFactor),
    scale: motionScale(motion, scaleFactor),
    shapes,
  });
}

function shapeLayer({ name, ind, frames, position, rotation = prop(0), scale = prop([100, 100, 100]), opacity = prop(100), shapes }) {
  return {
    ddd: 0,
    ind,
    ty: 4,
    nm: name,
    sr: 1,
    ks: {
      o: opacity,
      r: rotation,
      p: position,
      a: prop([0, 0, 0]),
      s: scale,
    },
    ao: 0,
    shapes,
    ip: 0,
    op: frames,
    st: 0,
    bm: 0,
  };
}

function motionPosition(base, motion, scaleFactor = 1) {
  return keyProp(motion.map((m) => ({
    t: m.t,
    v: [base[0] + m.x * scaleFactor, base[1] + m.y * scaleFactor, 0],
  })));
}

function motionRotation(motion, scaleFactor = 1) {
  return keyProp(motion.map((m) => ({ t: m.t, v: m.r * scaleFactor })));
}

function motionScale(motion, scaleFactor = 1) {
  return keyProp(motion.map((m) => {
    const amount = 100 + (m.s - 100) * scaleFactor;
    return { t: m.t, v: [amount, amount, 100] };
  }));
}

function eyeScaleProp(state, frames) {
  if (state === 'startled') {
    return keyProp([
      { t: 0, v: [100, 100, 100] },
      { t: frames * 0.18, v: [115, 115, 100] },
      { t: frames, v: [100, 100, 100] },
    ]);
  }
  const blinkAt = state === 'thinking' ? 23 : 62;
  return keyProp([
    { t: 0, v: [100, 100, 100] },
    { t: blinkAt, v: [100, 100, 100] },
    { t: blinkAt + 3, v: [100, 8, 100] },
    { t: blinkAt + 6, v: [100, 100, 100] },
    { t: frames, v: [100, 100, 100] },
  ]);
}

function pulseProp(min, max, frames) {
  return keyProp([
    { t: 0, v: min },
    { t: frames / 2, v: max },
    { t: frames, v: min },
  ]);
}

function prop(k) {
  return { a: 0, k };
}

function keyProp(points) {
  if (points.length === 1) return prop(points[0].v);
  return {
    a: 1,
    k: points.map((point, index) => {
      const next = points[index + 1];
      const key = {
        t: round(point.t),
        s: Array.isArray(point.v) ? point.v : [point.v],
        i: { x: [0.42], y: [1] },
        o: { x: [0.58], y: [0] },
      };
      if (next) key.e = Array.isArray(next.v) ? next.v : [next.v];
      return key;
    }),
  };
}

function group(items, name) {
  return {
    ty: 'gr',
    nm: name,
    it: [
      ...items,
      {
        ty: 'tr',
        p: prop([0, 0]),
        a: prop([0, 0]),
        s: prop([100, 100]),
        r: prop(0),
        o: prop(100),
        sk: prop(0),
        sa: prop(0),
      },
    ],
  };
}

function ellipse(position, size) {
  return {
    ty: 'el',
    nm: 'ellipse',
    p: prop(position),
    s: prop(size),
  };
}

function rect(position, size, radius = 0) {
  return {
    ty: 'rc',
    nm: 'rect',
    p: prop(position),
    s: prop(size),
    r: prop(radius),
  };
}

function path(vertices, closed) {
  return bezierPath(vertices, vertices.map(() => [0, 0]), vertices.map(() => [0, 0]), closed);
}

function bezierPath(vertices, inTangents, outTangents, closed) {
  return {
    ty: 'sh',
    nm: 'path',
    ks: {
      a: 0,
      k: {
        i: inTangents,
        o: outTangents,
        v: vertices,
        c: closed,
      },
    },
  };
}

function fill(color, opacity = 100) {
  return {
    ty: 'fl',
    nm: 'fill',
    c: prop(rgba(color)),
    o: prop(opacity),
    r: 1,
  };
}

function stroke(color, width, opacity = 100) {
  return {
    ty: 'st',
    nm: 'stroke',
    c: prop(rgba(color)),
    o: prop(opacity),
    w: prop(width),
    lc: 2,
    lj: 2,
    ml: 4,
  };
}

function rgba(hex) {
  const normalized = hex.replace('#', '');
  const value = Number.parseInt(normalized, 16);
  const r = (value >> 16) & 255;
  const g = (value >> 8) & 255;
  const b = value & 255;
  return [r / 255, g / 255, b / 255, 1];
}

function round(n) {
  return Math.round(n * 100) / 100;
}
