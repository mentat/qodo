import { useEffect, useState } from 'react';
import { SimpleGrid, Card, Text, Group, TextInput, Button, Title, Stack, Center, Loader } from '@mantine/core';
import { IconSearch } from '@tabler/icons-react';
import { useWeatherStore } from '../../store/weatherStore';

const ICON: Record<string, string> = {
  sun: '☀️',
  'cloud-sun': '⛅',
  cloud: '☁️',
  'cloud-rain': '🌧️',
  bolt: '⚡',
  haze: '🌫️',
  fog: '🌁',
};

export function WeatherApp() {
  const forecast = useWeatherStore((s) => s.forecast);
  const location = useWeatherStore((s) => s.location);
  const loading = useWeatherStore((s) => s.loading);
  const fetchForecast = useWeatherStore((s) => s.fetch);
  const [input, setInput] = useState(location);

  useEffect(() => {
    if (!forecast) void fetchForecast();
  }, [forecast, fetchForecast]);

  return (
    <Stack maw={900} mx="auto">
      <Group justify="space-between" wrap="wrap">
        <Title order={3}>🌆 Weather</Title>
        <Group gap="xs">
          <TextInput placeholder="City" value={input} onChange={(e) => setInput(e.currentTarget.value)} />
          <Button leftSection={<IconSearch size={16} />} onClick={() => void fetchForecast(input)}>
            Forecast
          </Button>
        </Group>
      </Group>

      {loading || !forecast ? (
        <Center mih={200}>
          <Loader />
        </Center>
      ) : (
        <>
          <Text c="dimmed" size="sm">
            Simulated forecast for {forecast.location}.
          </Text>
          <SimpleGrid cols={{ base: 2, sm: 3, md: 6 }}>
            {forecast.days.map((d) => (
              <Card
                key={d.date}
                withBorder
                radius="md"
                padding="md"
                style={{
                  textAlign: 'center',
                  background: 'linear-gradient(160deg, var(--mantine-color-synthPurple-light), var(--mantine-color-electricBlue-light))',
                }}
              >
                <Text size="sm" fw={700}>
                  {d.weekday.slice(0, 3)}
                </Text>
                <Text fz={32}>{ICON[d.icon] ?? '✨'}</Text>
                <Text size="sm">
                  {d.highC}° / {d.lowC}°
                </Text>
                <Text size="xs" c="dimmed">
                  {d.condition}
                </Text>
                <Text fz={11} c="dimmed">
                  💧 {d.precipPct}%
                </Text>
              </Card>
            ))}
          </SimpleGrid>
        </>
      )}
    </Stack>
  );
}
