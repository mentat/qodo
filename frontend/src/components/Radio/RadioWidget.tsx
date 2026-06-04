import { ActionIcon, Group, Text, Tooltip } from '@mantine/core';
import { IconPlayerPlayFilled, IconPlayerPauseFilled } from '@tabler/icons-react';
import { useRadioStore } from '../../store/radioStore';

// Compact "now playing" widget for the global Header. Mirrors the same
// useRadioStore the RadioApp uses, so play/pause from the header and from the
// Radio page stay in sync (single shared <audio> element via audioEngine).
// Hidden until the user has actually played something — keeps the header
// uncluttered for fresh sessions.
export function RadioWidget() {
  const current = useRadioStore((s) => s.current);
  const playing = useRadioStore((s) => s.playing);
  const toggle = useRadioStore((s) => s.toggle);

  if (!current) return null;

  return (
    <Tooltip label={`${current.title} — ${current.artist}`} openDelay={300}>
      <Group gap={6} wrap="nowrap" style={{ maxWidth: 200 }}>
        <Text
          size="sm"
          fw={500}
          truncate
          style={{ maxWidth: 140 }}
          aria-label="Now playing"
        >
          {current.title}
        </Text>
        <ActionIcon
          variant="default"
          size="lg"
          onClick={() => void toggle()}
          aria-label={playing ? 'Pause radio' : 'Play radio'}
        >
          {playing ? <IconPlayerPauseFilled size={18} /> : <IconPlayerPlayFilled size={18} />}
        </ActionIcon>
      </Group>
    </Tooltip>
  );
}
