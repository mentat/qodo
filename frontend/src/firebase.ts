import { initializeApp } from 'firebase/app';
import { getAuth, GoogleAuthProvider } from 'firebase/auth';
import { getFirestore } from 'firebase/firestore';

const firebaseConfig = {
  apiKey: 'AIzaSyAJi2ig_BvcjV8BOC_vrK0abIZ2usjHH4o',
  authDomain: 'qodo-demo.firebaseapp.com',
  projectId: 'qodo-demo',
  storageBucket: 'qodo-demo.firebasestorage.app',
  messagingSenderId: '600919524846',
  appId: '1:600919524846:web:8ce1aa37e11dcea658faff',
};

const app = initializeApp(firebaseConfig);
export const auth = getAuth(app);
export const googleProvider = new GoogleAuthProvider();
// Firestore client — used for live (onSnapshot) reads of emails + events.
// Writes still go through the Go API (Admin SDK); these client reads are
// owner-scoped by firestore.rules.
export const db = getFirestore(app);
