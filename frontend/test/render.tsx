import type { ReactNode } from 'react';
import { render } from '@testing-library/react';
import { MantineProvider } from '@mantine/core';
import { theme } from '../src/theme';

// Mantine components throw without a provider; wrap every render in one.
export function renderWithMantine(ui: ReactNode) {
  return render(<MantineProvider theme={theme}>{ui}</MantineProvider>);
}
