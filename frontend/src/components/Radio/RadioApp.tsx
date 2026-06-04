import { useEffect } from 'react';
import { Stack, Group, Text, ActionIcon, Paper, Title } from '@mantine/core';
import { IconPlayerPlayFilled, IconPlayerPauseFilled } from '@tabler/icons-react';
import { useRadioStore } from '../../store/radioStore';
import { SpectrumAnalyzer } from '../SpectrumAnalyzer';

export function RadioApp() {
  const tracks = useRadioStore((s) => s.tracks);
  const current = useRadioStore((s) => s.current);
  const playing = useRadioStore((s) => s.playing);
  const load = useRadioStore((s) => s.load);
  const play = useRadioStore((s) => s.play);
  const toggle = useRadioStore((s) => s.toggle);

  useEffect(() => {
    void load();
  }, [load]);

  // Audio-element <-> store sync lives in App.tsx (one global subscription),
  // so the header widget stays accurate even when this page isn't mounted.

  return (
    <Stack maw={680} mx="auto">
      <Title order={3}>📻 Synthwave Radio</Title>
      <Paper withBorder radius="md" p="md">
        <SpectrumAnalyzer thinking={false} />
        <Group justify="space-between" mt="sm">
          <div>
            <Text fw={700}>{current ? current.title : 'Nothing playing'}</Text>
            <Text size="sm" c="dimmed">
              {current ? current.artist : 'Press play to start the night drive'}
            </Text>
          </div>
          <ActionIcon size={48} radius="xl" variant="filled" onClick={() => void toggle()} aria-label="Play/pause">
            {playing ? <IconPlayerPauseFilled size={24} /> : <IconPlayerPlayFilled size={24} />}
          </ActionIcon>
        </Group>
      </Paper>

      <Stack gap={4}>
        {tracks.map((t) => {
          const isCurrent = current?.id === t.id;
          return (
            <Group
              key={t.id}
              justify="space-between"
              p="xs"
              style={{ borderRadius: 8, background: isCurrent ? 'var(--mantine-color-synthPurple-light)' : 'transparent' }}
            >
              <div>
                <Text size="sm" fw={isCurrent ? 700 : 500}>
                  {t.title}
                </Text>
                <Text size="xs" c="dimmed">
                  {t.artist}
                </Text>
              </div>
              <ActionIcon variant="subtle" onClick={() => void (isCurrent ? toggle() : play(t))} aria-label="Play track">
                {isCurrent && playing ? <IconPlayerPauseFilled size={18} /> : <IconPlayerPlayFilled size={18} />}
              </ActionIcon>
            </Group>
          );
        })}
      </Stack>
    </Stack>
  );
}
