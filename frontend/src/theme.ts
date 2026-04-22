import { createTheme, type MantineColorsTuple } from '@mantine/core';

// 80s New Wave neon palette
const neonPink: MantineColorsTuple = [
  '#ffe6f7',
  '#ffccee',
  '#ff99dd',
  '#ff66cc',
  '#ff33bb',
  '#ff00aa',
  '#cc0088',
  '#990066',
  '#660044',
  '#330022',
];

const electricBlue: MantineColorsTuple = [
  '#e6f0ff',
  '#b3d4ff',
  '#80b8ff',
  '#4d9cff',
  '#1a80ff',
  '#0066ff',
  '#0052cc',
  '#003d99',
  '#002966',
  '#001433',
];

const synthPurple: MantineColorsTuple = [
  '#f0e6ff',
  '#d4b3ff',
  '#b880ff',
  '#9c4dff',
  '#7f1aff',
  '#6600ff',
  '#5200cc',
  '#3d0099',
  '#290066',
  '#140033',
];

const neonGreen: MantineColorsTuple = [
  '#e6ffe6',
  '#b3ffb3',
  '#66ff66',
  '#33ff33',
  '#00ff00',
  '#00cc00',
  '#009900',
  '#006600',
  '#003300',
  '#001a00',
];

const hotYellow: MantineColorsTuple = [
  '#fffde6',
  '#fff9b3',
  '#fff580',
  '#fff14d',
  '#ffed1a',
  '#ffe600',
  '#ccb800',
  '#998a00',
  '#665c00',
  '#332e00',
];

export const theme = createTheme({
  primaryColor: 'synthPurple',
  colors: {
    neonPink,
    electricBlue,
    synthPurple,
    neonGreen,
    hotYellow,
  },
  fontFamily: '"Inter", "Segoe UI", system-ui, -apple-system, sans-serif',
  headings: {
    fontFamily: '"Inter", "Segoe UI", system-ui, sans-serif',
    fontWeight: '800',
  },
  radius: {
    xs: '4px',
    sm: '6px',
    md: '10px',
    lg: '16px',
    xl: '24px',
  },
  defaultRadius: 'md',
  other: {
    style: '80s-new-wave',
  },
});
