import { useMemo } from 'react';
import { Stack, Tooltip, UnstyledButton, Text, Indicator } from '@mantine/core';
import {
  IconChecklist,
  IconMail,
  IconCalendar,
  IconAddressBook,
  IconNotes,
  IconRadio,
  IconCloud,
  type Icon,
} from '@tabler/icons-react';
import { useUIStore, type AppId } from '../store/uiStore';
import { useMailStore, totalUnread } from '../store/mailStore';

const APPS: { id: AppId; label: string; Icon: Icon }[] = [
  { id: 'todos', label: 'Todos', Icon: IconChecklist },
  { id: 'mail', label: 'Mail', Icon: IconMail },
  { id: 'calendar', label: 'Calendar', Icon: IconCalendar },
  { id: 'contacts', label: 'Contacts', Icon: IconAddressBook },
  { id: 'notes', label: 'Notes', Icon: IconNotes },
  { id: 'radio', label: 'Radio', Icon: IconRadio },
  { id: 'weather', label: 'Weather', Icon: IconCloud },
];

export function NavRail() {
  const activeApp = useUIStore((s) => s.activeApp);
  const setActiveApp = useUIStore((s) => s.setActiveApp);
  const emails = useMailStore((s) => s.emails);
  const unread = useMemo(() => totalUnread(emails), [emails]);

  return (
    <Stack gap={6} p="xs" align="stretch">
      {APPS.map(({ id, label, Icon }) => {
        const active = id === activeApp;
        const button = (
          <UnstyledButton
            onClick={() => setActiveApp(id)}
            aria-label={label}
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: 2,
              padding: '8px 4px',
              borderRadius: 10,
              color: active ? 'var(--mantine-color-white)' : 'var(--mantine-color-dimmed)',
              background: active
                ? 'linear-gradient(135deg, var(--mantine-color-synthPurple-6), var(--mantine-color-neonPink-6))'
                : 'transparent',
              transition: 'background 120ms ease',
            }}
          >
            <Icon size={22} stroke={1.6} />
            <Text size="10px" fw={active ? 700 : 500}>
              {label}
            </Text>
          </UnstyledButton>
        );
        return (
          <Tooltip key={id} label={label} position="right" withArrow>
            {id === 'mail' && unread > 0 ? (
              <Indicator label={unread} size={16} color="neonPink" offset={6}>
                {button}
              </Indicator>
            ) : (
              button
            )}
          </Tooltip>
        );
      })}
    </Stack>
  );
}
