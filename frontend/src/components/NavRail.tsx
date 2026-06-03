import { useMemo } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Stack, Tooltip, UnstyledButton, Text, Badge } from '@mantine/core';
import {
  IconChecklist,
  IconMail,
  IconCalendar,
  IconAddressBook,
  IconNotes,
  IconRadio,
  IconCloud,
  IconWorld,
  type Icon,
} from '@tabler/icons-react';
import { useMailStore, totalUnread } from '../store/mailStore';

const APPS: { path: string; label: string; Icon: Icon }[] = [
  { path: '/todos', label: 'Todos', Icon: IconChecklist },
  { path: '/mail', label: 'Mail', Icon: IconMail },
  { path: '/calendar', label: 'Calendar', Icon: IconCalendar },
  { path: '/contacts', label: 'Contacts', Icon: IconAddressBook },
  { path: '/notes', label: 'Notes', Icon: IconNotes },
  { path: '/radio', label: 'Radio', Icon: IconRadio },
  { path: '/weather', label: 'Weather', Icon: IconCloud },
  { path: '/risk', label: 'Risk', Icon: IconWorld },
];

export function NavRail() {
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const emails = useMailStore((s) => s.emails);
  const unread = useMemo(() => totalUnread(emails), [emails]);

  return (
    <Stack gap={6} p="xs" align="stretch">
      {APPS.map(({ path, label, Icon }) => {
        const active = pathname === path || (path === '/todos' && pathname === '/');
        return (
          <Tooltip key={path} label={label} position="right" withArrow>
            <UnstyledButton
              onClick={() => navigate(path)}
              aria-label={label}
              aria-current={active ? 'page' : undefined}
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
              {path === '/mail' && unread > 0 && (
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
