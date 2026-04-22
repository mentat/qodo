import { initializeApp } from 'firebase/app';
import { getAuth, GoogleAuthProvider } from 'firebase/auth';

const firebaseConfig = {
  apiKey: 'AIzaSyAJi2ig_BvcjV8BOC_vrK0abIZ2usjHH4o',
  authDomain: 'qodo-demo.firebaseapp.com',
  projectId: 'qodo-demo',
  storageBucket: 'qodo-demo.firebasestorage.app',
  messagingSenderId: '600919524846',
  appId: '1:600919524846:web:c664b3e04ea6bde758faff',
};

const app = initializeApp(firebaseConfig);
export const auth = getAuth(app);
export const googleProvider = new GoogleAuthProvider();
