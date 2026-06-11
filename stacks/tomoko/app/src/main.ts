import { mount } from 'svelte';
import App from './App.svelte';
import '@typopro/web-fantasque-sans-mono/TypoPRO-FantasqueSansMono.css';
import '@typopro/web-fantasque-sans-mono/TypoPRO-FantasqueSansMono-Bold.css';
import './app.css';

const app = mount(App, {
  target: document.getElementById('app')!,
});

export default app;
