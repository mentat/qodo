import { useEffect } from 'react';
import { Stack, Group, Text, ActionIcon, Title, Badge } from '@mantine/core';
import {
  IconPlayerPauseFilled,
  IconPlayerPlayFilled,
  IconPlayerStopFilled,
  IconPlayerTrackNextFilled,
  IconPlayerTrackPrevFilled,
} from '@tabler/icons-react';
import { useRadioStore } from '../../store/radioStore';
import type { Track } from '../../types/radio';

function getCurrentIndex(tracks: Track[], current: Track | null): number {
  return current ? tracks.findIndex((track) => track.id === current.id) : -1;
}

export function RadioApp() {
  const tracks = useRadioStore((s) => s.tracks);
  const current = useRadioStore((s) => s.current);
  const playing = useRadioStore((s) => s.playing);
  const load = useRadioStore((s) => s.load);
  const play = useRadioStore((s) => s.play);
  const toggle = useRadioStore((s) => s.toggle);
  const pause = useRadioStore((s) => s.pause);

  useEffect(() => {
    void load();
  }, [load]);

  // Audio-element <-> store sync lives in App.tsx (one global subscription),
  // so the header widget stays accurate even when this page isn't mounted.
  const currentIndex = getCurrentIndex(tracks, current);
  const hasTracks = tracks.length > 0;

  const playPrevious = async () => {
    if (!hasTracks) return;
    const previousIndex = currentIndex <= 0 ? tracks.length - 1 : currentIndex - 1;
    await play(tracks[previousIndex]);
  };

  const playNext = async () => {
    if (!hasTracks) return;
    const nextIndex = currentIndex < 0 ? 0 : (currentIndex + 1) % tracks.length;
    await play(tracks[nextIndex]);
  };

  return (
    <Stack className="radio-page" maw={1080} mx="auto" gap="lg">
      <Group justify="space-between" align="end" gap="sm">
        <div>
          <Text className="radio-kicker" size="xs" fw={800}>
            Portable Stereo Cassette
          </Text>
          <Title order={2} className="radio-title">
            Synthwave Radio
          </Title>
        </div>
        <Badge className={playing ? 'radio-live-badge is-live' : 'radio-live-badge'} variant="filled">
          {playing ? 'On Air' : 'Standby'}
        </Badge>
      </Group>

      <div className="radio-stage">
        <section className={playing ? 'radio-boombox is-playing' : 'radio-boombox'} aria-label="80s boombox radio">
          <div className="radio-boombox-handle" aria-hidden="true" />
          <div className="radio-boombox-face">
            <div className="radio-top-row">
              <div className="radio-brand">
                <span>SYNTH</span>
                <span>MAX</span>
              </div>
              <div className="radio-tuner">
                <TunerDial />
              </div>
            </div>

            <div className="radio-deck-row">
              <SpeakerGrill side="left" />
              <CassetteDeck current={current} playing={playing} />
              <SpeakerGrill side="right" />
            </div>

            <Group className="radio-control-deck" justify="center" gap="xs" wrap="nowrap">
              <ActionIcon
                className="radio-deck-button"
                size={42}
                variant="filled"
                radius="xs"
                aria-label="Previous track"
                disabled={!hasTracks}
                onClick={() => void playPrevious()}
              >
                <IconPlayerTrackPrevFilled size={20} />
              </ActionIcon>
              <ActionIcon
                className="radio-deck-button radio-deck-button-primary"
                size={52}
                variant="filled"
                radius="xs"
                aria-label={playing ? 'Pause radio' : 'Play radio'}
                disabled={!hasTracks}
                onClick={() => void toggle()}
              >
                {playing ? <IconPlayerPauseFilled size={25} /> : <IconPlayerPlayFilled size={25} />}
              </ActionIcon>
              <ActionIcon
                className="radio-deck-button"
                size={42}
                variant="filled"
                radius="xs"
                aria-label="Stop radio"
                disabled={!current}
                onClick={pause}
              >
                <IconPlayerStopFilled size={18} />
              </ActionIcon>
              <ActionIcon
                className="radio-deck-button"
                size={42}
                variant="filled"
                radius="xs"
                aria-label="Next track"
                disabled={!hasTracks}
                onClick={() => void playNext()}
              >
                <IconPlayerTrackNextFilled size={20} />
              </ActionIcon>
            </Group>
          </div>
        </section>

        <TapeStack
          tracks={tracks}
          current={current}
          playing={playing}
          onSelect={(track) => void (current?.id === track.id ? toggle() : play(track))}
        />
      </div>
    </Stack>
  );
}

function SpeakerGrill({ side }: { side: 'left' | 'right' }) {
  return (
    <div className={`radio-speaker radio-speaker-${side}`} aria-hidden="true">
      <div className="radio-speaker-ring">
        <div className="radio-speaker-core" />
      </div>
    </div>
  );
}

function TapeStack({
  tracks,
  current,
  playing,
  onSelect,
}: {
  tracks: Track[];
  current: Track | null;
  playing: boolean;
  onSelect: (track: Track) => void;
}) {
  return (
    <aside className="radio-tape-stack" aria-label="Cassette tape library">
      <Text className="radio-tape-stack-title" size="xs" fw={900}>
        Tape Library
      </Text>
      <Stack gap={8}>
        {tracks.length === 0 && (
          <div className="radio-tape-card is-empty">
            <span className="radio-tape-spine" aria-hidden="true" />
            <div className="radio-tape-label">
              <Text component="span" size="sm" fw={800}>
                Loading tapes
              </Text>
              <Text component="span" size="xs">Please stand by</Text>
            </div>
          </div>
        )}

        {tracks.map((track, index) => {
          const isCurrent = current?.id === track.id;
          return (
            <button
              key={track.id}
              type="button"
              className={isCurrent ? 'radio-tape-card is-current' : 'radio-tape-card'}
              aria-label={isCurrent && playing ? `Pause ${track.title}` : `Play ${track.title}`}
              aria-pressed={isCurrent}
              onClick={() => onSelect(track)}
            >
              <span className="radio-tape-spine" aria-hidden="true">
                {String(index + 1).padStart(2, '0')}
              </span>
              <span className="radio-tape-label">
                <Text component="span" size="sm" fw={900} truncate>
                  {track.title}
                </Text>
                <Text component="span" size="xs" truncate>
                  {track.artist}
                </Text>
              </span>
              <span className="radio-tape-reels" aria-hidden="true">
                <span />
                <span />
              </span>
              <span className="radio-tape-play-state" aria-hidden="true">
                {isCurrent && playing ? <IconPlayerPauseFilled size={16} /> : <IconPlayerPlayFilled size={16} />}
              </span>
            </button>
          );
        })}
      </Stack>
    </aside>
  );
}

function TunerDial() {
  const fmMarks = ['88', '92', '96', '100', '104', '108'];
  const amMarks = ['530', '700', '900', '1100', '1300', '1600'];
  const ticks = Array.from({ length: 25 }, (_, index) => index);

  return (
    <div className="radio-tuner-dial" aria-label="AM/FM tuner dial">
      <div className="radio-tuner-glass">
        <div className="radio-tuner-row radio-tuner-row-fm">
          <Text className="radio-band-label" size="xs" fw={900}>
            FM MHz
          </Text>
          <div className="radio-tuner-track">
            <div className="radio-tuner-ticks" aria-hidden="true">
              {ticks.map((tick) => (
                <span key={`fm-${tick}`} className={tick % 4 === 0 ? 'is-major' : ''} />
              ))}
            </div>
            <div className="radio-tuner-numbers" aria-hidden="true">
              {fmMarks.map((mark) => (
                <span key={mark}>{mark}</span>
              ))}
            </div>
          </div>
        </div>

        <div className="radio-tuner-row">
          <Text className="radio-band-label" size="xs" fw={900}>
            AM kHz
          </Text>
          <div className="radio-tuner-track">
            <div className="radio-tuner-ticks" aria-hidden="true">
              {ticks.map((tick) => (
                <span key={`am-${tick}`} className={tick % 4 === 0 ? 'is-major' : ''} />
              ))}
            </div>
            <div className="radio-tuner-numbers" aria-hidden="true">
              {amMarks.map((mark) => (
                <span key={mark}>{mark}</span>
              ))}
            </div>
          </div>
        </div>

        <div className="radio-tuner-needle" aria-hidden="true" />
        <div className="radio-signal-meter" aria-hidden="true">
          <span />
          <span />
          <span />
          <span className="is-hot" />
        </div>
      </div>
    </div>
  );
}

function CassetteDeck({ current, playing }: { current: Track | null; playing: boolean }) {
  return (
    <div className="radio-cassette-wrap">
      <svg
        className={playing ? 'radio-cassette is-playing' : 'radio-cassette'}
        data-testid="radio-cassette"
        viewBox="0 0 420 245"
        role="img"
        aria-label={playing ? 'Cassette tape playing' : 'Cassette tape stopped'}
      >
        <defs>
          <linearGradient id="cassetteBody" x1="0" x2="1" y1="0" y2="1">
            <stop offset="0%" stopColor="#f6c5e5" />
            <stop offset="50%" stopColor="#f48ab6" />
            <stop offset="100%" stopColor="#4ee0df" />
          </linearGradient>
          <linearGradient id="cassetteWindow" x1="0" x2="1" y1="0" y2="1">
            <stop offset="0%" stopColor="#221522" />
            <stop offset="100%" stopColor="#05070f" />
          </linearGradient>
        </defs>

        <rect x="16" y="18" width="388" height="209" rx="18" fill="url(#cassetteBody)" />
        <rect x="31" y="33" width="358" height="179" rx="12" fill="#181823" opacity="0.2" />
        <rect x="58" y="48" width="304" height="52" rx="7" fill="#fff6d7" />
        <rect x="85" y="62" width="250" height="10" rx="5" fill="#f0629f" opacity="0.45" />
        <rect x="118" y="80" width="184" height="9" rx="4.5" fill="#3acbd5" opacity="0.75" />

        <rect x="78" y="116" width="264" height="77" rx="16" fill="url(#cassetteWindow)" />
        <path d="M139 154 C174 132 247 132 282 154" fill="none" stroke="#d28050" strokeWidth="9" strokeLinecap="round" />
        <path d="M139 154 C174 176 247 176 282 154" fill="none" stroke="#6e3a22" strokeWidth="7" strokeLinecap="round" opacity="0.8" />

        <g className="radio-tape-wheel radio-tape-wheel-left">
          <circle cx="139" cy="154" r="35" fill="#12121c" />
          <circle cx="139" cy="154" r="26" fill="#322236" />
          {[0, 60, 120, 180, 240, 300].map((rotation) => (
            <rect
              key={rotation}
              x="136"
              y="126"
              width="6"
              height="18"
              rx="3"
              fill="#f4eec6"
              transform={`rotate(${rotation} 139 154)`}
            />
          ))}
          <circle cx="139" cy="154" r="8" fill="#07070c" />
        </g>

        <g className="radio-tape-wheel radio-tape-wheel-right">
          <circle cx="282" cy="154" r="35" fill="#12121c" />
          <circle cx="282" cy="154" r="26" fill="#322236" />
          {[0, 60, 120, 180, 240, 300].map((rotation) => (
            <rect
              key={rotation}
              x="279"
              y="126"
              width="6"
              height="18"
              rx="3"
              fill="#f4eec6"
              transform={`rotate(${rotation} 282 154)`}
            />
          ))}
          <circle cx="282" cy="154" r="8" fill="#07070c" />
        </g>

        <rect x="153" y="202" width="112" height="18" rx="6" fill="#15151d" />
        <circle cx="62" cy="52" r="6" fill="#12121c" />
        <circle cx="358" cy="52" r="6" fill="#12121c" />
        <circle cx="62" cy="196" r="6" fill="#12121c" />
        <circle cx="358" cy="196" r="6" fill="#12121c" />
      </svg>
      <div className="radio-cassette-label">
        <Text size="xs" fw={900} truncate>
          {current ? current.title : 'Insert mixtape'}
        </Text>
        <Text size="xs" truncate>
          {current ? current.artist : 'Press play to start'}
        </Text>
      </div>
    </div>
  );
}
