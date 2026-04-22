import { useState } from 'react';
import {
  Paper,
  TextInput,
  PasswordInput,
  Button,
  Title,
  Text,
  Divider,
  Stack,
  Container,
  Center,
  Alert,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { IconBrandGoogle, IconAlertCircle } from '@tabler/icons-react';
import { useAuth } from '../hooks/useAuth';

export function LoginPage() {
  const { signIn, signUp, signInWithGoogle } = useAuth();
  const [mode, setMode] = useState<'login' | 'signup'>('login');
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const form = useForm({
    initialValues: { email: '', password: '' },
    validate: {
      email: (v) => (/^\S+@\S+$/.test(v) ? null : 'Invalid email'),
      password: (v) => (v.length >= 6 ? null : 'Password must be at least 6 characters'),
    },
  });

  const handleSubmit = async (values: typeof form.values) => {
    setError(null);
    setLoading(true);
    try {
      if (mode === 'login') {
        await signIn(values.email, values.password);
      } else {
        await signUp(values.email, values.password);
      }
    } catch (e: any) {
      setError(e.message?.replace('Firebase: ', '') || 'Authentication failed');
    } finally {
      setLoading(false);
    }
  };

  const handleGoogle = async () => {
    setError(null);
    try {
      await signInWithGoogle();
    } catch (e: any) {
      setError(e.message?.replace('Firebase: ', '') || 'Google sign-in failed');
    }
  };

  return (
    <Center mih="100vh" bg="var(--mantine-color-body)">
      <Container size={420} w="100%">
        <Title ta="center" mb="md">
          Qodo TODO
        </Title>
        <Text c="dimmed" size="sm" ta="center" mb="xl">
          {mode === 'login' ? "Don't have an account? " : 'Already have an account? '}
          <Text
            component="a"
            href="#"
            size="sm"
            c="blue"
            onClick={(e) => {
              e.preventDefault();
              setMode(mode === 'login' ? 'signup' : 'login');
              setError(null);
            }}
          >
            {mode === 'login' ? 'Create one' : 'Sign in'}
          </Text>
        </Text>

        <Paper withBorder shadow="md" p={30} radius="md">
          {error && (
            <Alert icon={<IconAlertCircle size={16} />} color="red" mb="md" variant="light">
              {error}
            </Alert>
          )}

          <form onSubmit={form.onSubmit(handleSubmit)}>
            <Stack>
              <TextInput
                label="Email"
                placeholder="you@example.com"
                required
                {...form.getInputProps('email')}
              />
              <PasswordInput
                label="Password"
                placeholder="Your password"
                required
                {...form.getInputProps('password')}
              />
              <Button type="submit" fullWidth loading={loading}>
                {mode === 'login' ? 'Sign in' : 'Create account'}
              </Button>
            </Stack>
          </form>

          <Divider label="Or continue with" labelPosition="center" my="lg" />

          <Button
            fullWidth
            variant="default"
            leftSection={<IconBrandGoogle size={16} />}
            onClick={handleGoogle}
          >
            Google
          </Button>
        </Paper>
      </Container>
    </Center>
  );
}
