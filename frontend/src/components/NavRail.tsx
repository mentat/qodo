import { useMemo } from 'react';
import { Stack, Tooltip, UnstyledButton, Text, Badge } from '@mantine/core';
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
        return (
          <Tooltip key={id} label={label} position="right" withArrow>
            <UnstyledButton
              onClick={() => setActiveApp(id)}
              aria-label={label}
              style={{
                position: 'relative',
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
              {id === 'mail' && unread > 0 && (
                <Badge
                  size="sm"
                  circle={unread <= 9}
                  color="neonPink"
                  style={{ position: 'absolute', top: 0, right: 12, pointerEvents: 'none' }}
                >
                  {unread > 99 ? '99+' : unread}
                </Badge>
              )}
            </UnstyledButton>
          </Tooltip>
        );
      })}
    </Stack>
  );
}
