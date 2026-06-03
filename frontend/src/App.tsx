import { useEffect } from 'react';
import { Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { AppShell, Center, Loader } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useAuth } from './hooks/useAuth';
import { LoginPage } from './components/LoginPage';
import { Header } from './components/Header';
import { NavRail } from './components/NavRail';
import { ChatPanel } from './components/ChatPanel';
import { TodosApp } from './components/Todos/TodosApp';
import { MailApp } from './components/Mail/MailApp';
import { CalendarApp } from './components/Calendar/CalendarApp';
import { ContactsApp } from './components/Contacts/ContactsApp';
import { NotesApp } from './components/Notes/NotesApp';
import { RadioApp } from './components/Radio/RadioApp';
import { WeatherApp } from './components/Weather/WeatherApp';
import { RiskApp } from './components/Risk/RiskApp';
import { useUIStore } from './store/uiStore';
import { useTodoStore } from './store/todoStore';
import { useMailStore } from './store/mailStore';
import { useEventStore } from './store/eventStore';
import { seedDemo, resetDemo } from './api/demo';

const TITLES: Record<string, string> = {
  '/todos': 'Todos',
  '/mail': 'Mail',
  '/calendar': 'Calendar',
  '/contacts': 'Contacts',
  '/notes': 'Notes',
  '/radio': 'Radio',
  '/weather': 'Weather',
  '/risk': 'Risk',
};

export default function App() {
  const { user, loading: authLoading, signOut } = useAuth();
  const { pathname } = useLocation();
  const chatOpen = useUIStore((s) => s.chatOpen);
  const openChat = useUIStore((s) => s.openChat);
  const closeChat = useUIStore((s) => s.closeChat);
  const fetchTodos = useTodoStore((s) => s.fetchTodos);
  const subscribeMail = useMailStore((s) => s.subscribe);
  const unsubscribeMail = useMailStore((s) => s.unsubscribe);
  const subscribeEvents = useEventStore((s) => s.subscribe);
  const unsubscribeEvents = useEventStore((s) => s.unsubscribe);

  // On login: seed demo content (idempotent), load todos, and open the live
  // Firestore listeners for mail + calendar. Tear the listeners down on logout.
  useEffect(() => {
    if (!user) return;
    const uid = user.uid;
    void fetchTodos();
    seedDemo().catch((e) => console.error('seed failed', e));
    subscribeMail(uid);
    subscribeEvents(uid);
    return () => {
      unsubscribeMail();
      unsubscribeEvents();
    };
  }, [user, fetchTodos, subscribeMail, unsubscribeMail, subscribeEvents, unsubscribeEvents]);

  const handleReset = async () => {
    try {
      await resetDemo();
      await fetchTodos();
      notifications.show({ title: 'Reset', message: 'Demo data restored. BZZT.', color: 'synthPurple' });
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    }
  };

  if (authLoading) {
    return (
      <Center mih="100vh">
        <Loader size="lg" />
      </Center>
    );
  }
  if (!user) return <LoginPage />;

  return (
    <AppShell header={{ height: 60 }} navbar={{ width: 84, breakpoint: 0 }} padding="md">
      <AppShell.Header>
        <Header
          user={user}
          title={TITLES[pathname] ?? 'Synthwave OS'}
          onSignOut={signOut}
          onOpenChat={openChat}
          onResetDemo={handleReset}
        />
      </AppShell.Header>

      <AppShell.Navbar>
        <NavRail />
      </AppShell.Navbar>

      <AppShell.Main>
        <Routes>
          <Route path="/" element={<Navigate to="/todos" replace />} />
          <Route path="/todos" element={<TodosApp />} />
          <Route path="/mail" element={<MailApp />} />
          <Route path="/calendar" element={<CalendarApp />} />
          <Route path="/contacts" element={<ContactsApp />} />
          <Route path="/notes" element={<NotesApp />} />
          <Route path="/radio" element={<RadioApp />} />
          <Route path="/weather" element={<WeatherApp />} />
          <Route path="/risk" element={<RiskApp />} />
          <Route path="*" element={<Navigate to="/todos" replace />} />
        </Routes>
        <ChatPanel opened={chatOpen} onClose={closeChat} />
      </AppShell.Main>
    </AppShell>
  );
}
