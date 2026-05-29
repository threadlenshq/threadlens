import { mount } from 'svelte';
import './styles/tokens.css';
import ComponentPreview from './ComponentPreview.svelte';

const app = mount(ComponentPreview, {
  target: document.getElementById('preview'),
});

export default app;
