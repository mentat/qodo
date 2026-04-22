import {
  Group,
  Title,
  TextInput,
  ActionIcon,
  Menu,
  Avatar,
  useMantineColorScheme,
} from '@mantine/core';
import { useDebouncedValue } from '@mantine/hooks';
import { IconSearch, IconSun, IconMoon, IconLogout } from '@tabler/icons-react';
import { useState, useEffect } from 'react';
import type { User } from 'firebase/auth';

interface HeaderProps {
  user: User;
  onSearch: (query: string) => void;
  onSignOut: () => void;
}

export function Header({ user, onSearch, onSignOut }: HeaderProps) {
  const { colorScheme, toggleColorScheme } = useMantineColorScheme();
  const [search, setSearch] = useState('');
  const [debounced] = useDebouncedValue(search, 300);

  useEffect(() => {
    onSearch(debounced);
  }, [debounced, onSearch]);

  return (
    <Group h="100%" px="md" justify="space-between">
      <Group gap="xs">
        <img src="/logo.svg" alt="Qodo" height={28} />
        <Title order={3}>TODO</Title>
      </Group>

      <Group>
        <TextInput
          placeholder="Search todos..."
          leftSection={<IconSearch size={16} />}
          value={search}
          onChange={(e) => setSearch(e.currentTarget.value)}
          w={250}
        />

        <ActionIcon
          variant="default"
          size="lg"
          onClick={() => toggleColorScheme()}
          aria-label="Toggle color scheme"
        >
          {colorScheme === 'dark' ? <IconSun size={18} /> : <IconMoon size={18} />}
        </ActionIcon>

        <Menu shadow="md" width={200}>
          <Menu.Target>
            <ActionIcon variant="default" size="lg" radius="xl">
              <Avatar
                src={user.photoURL}
                alt={user.displayName || user.email || ''}
                size="sm"
                radius="xl"
              >
                {(user.displayName || user.email || '?')[0].toUpperCase()}
              </Avatar>
            </ActionIcon>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Label>{user.email}</Menu.Label>
            <Menu.Item leftSection={<IconLogout size={14} />} onClick={onSignOut}>
              Sign out
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Group>
    </Group>
  );
}
