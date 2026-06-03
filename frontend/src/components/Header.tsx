import { Group, Title, ActionIcon, Menu, Avatar, useMantineColorScheme } from '@mantine/core';
import { IconSun, IconMoon, IconLogout, IconRobotFace, IconRefresh } from '@tabler/icons-react';
import type { User } from 'firebase/auth';

interface HeaderProps {
  user: User;
  title: string;
  onSignOut: () => void;
  onOpenChat: () => void;
  onResetDemo: () => void;
}

export function Header({ user, title, onSignOut, onOpenChat, onResetDemo }: HeaderProps) {
  const { colorScheme, toggleColorScheme } = useMantineColorScheme();

  return (
    <Group h="100%" px="md" justify="space-between">
      <Group gap="xs">
        <img src="/logo.svg" alt="Qodo" height={28} />
        <Title order={4}>
          Synthwave OS · {title}
        </Title>
      </Group>

      <Group>
        <ActionIcon
          variant="filled"
          color="synthPurple"
          size="lg"
          onClick={onOpenChat}
          aria-label="Talk to Marvin"
          title="Talk to Marvin"
        >
          <IconRobotFace size={20} />
        </ActionIcon>

        <ActionIcon
          variant="default"
          size="lg"
          onClick={() => toggleColorScheme()}
          aria-label="Toggle color scheme"
        >
          {colorScheme === 'dark' ? <IconSun size={18} /> : <IconMoon size={18} />}
        </ActionIcon>

        <Menu shadow="md" width={220}>
          <Menu.Target>
            <ActionIcon variant="default" size="lg" radius="xl">
              <Avatar src={user.photoURL} alt={user.displayName || user.email || ''} size="sm" radius="xl">
                {(user.displayName || user.email || '?')[0].toUpperCase()}
              </Avatar>
            </ActionIcon>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Label>{user.email}</Menu.Label>
            <Menu.Item leftSection={<IconRefresh size={14} />} onClick={onResetDemo}>
              Reset demo data
            </Menu.Item>
            <Menu.Item leftSection={<IconLogout size={14} />} onClick={onSignOut}>
              Sign out
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Group>
    </Group>
  );
}
