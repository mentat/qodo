import { useEffect, type ReactNode } from 'react';
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
import { useUIStore, type AppId } from './store/uiStore';
import { useTodoStore } from './store/todoStore';
import { useMailStore } from './store/mailStore';
import { useEventStore } from './store/eventStore';
import { seedDemo, resetDemo } from './api/demo';

const APP_TITLES: Record<AppId, string> = {
  todos: 'Todos',
  mail: 'Mail',
  calendar: 'Calendar',
  contacts: 'Contacts',
  notes: 'Notes',
  radio: 'Radio',
  weather: 'Weather',
};

export default function App() {
  const { user, loading: authLoading, signOut } = useAuth();
  const activeApp = useUIStore((s) => s.activeApp);
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

  const screen: Record<AppId, ReactNode> = {
    todos: <TodosApp />,
    mail: <MailApp />,
    calendar: <CalendarApp />,
    contacts: <ContactsApp />,
    notes: <NotesApp />,
    radio: <RadioApp />,
    weather: <WeatherApp />,
  };

  return (
    <AppShell header={{ height: 60 }} navbar={{ width: 84, breakpoint: 0 }} padding="md">
      <AppShell.Header>
        <Header
          user={user}
          title={APP_TITLES[activeApp]}
          onSignOut={signOut}
          onOpenChat={openChat}
          onResetDemo={handleReset}
        />
      </AppShell.Header>

      <AppShell.Navbar>
        <NavRail />
      </AppShell.Navbar>

      <AppShell.Main>
        {screen[activeApp]}
        <ChatPanel opened={chatOpen} onClose={closeChat} />
      </AppShell.Main>
    </AppShell>
  );
}
