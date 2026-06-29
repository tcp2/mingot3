<script>
    import { createEventDispatcher } from "svelte";
    const dispatch = createEventDispatcher();
</script>

<header class="glass-header" style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
    <div class="logo" style="display: flex; align-items: center; gap: 8px;">
        <i class="fa-solid fa-ghost"></i>
        <h1 style="font-size: 1.25rem; margin: 0;">Mingot</h1>
    </div>
    <div class="header-actions">
        <button 
            class="btn-icon" 
            style="background: transparent; border: none; color: var(--text-color); font-size: 1.15rem; cursor: pointer; padding: 8px; border-radius: 50%; display: flex; align-items: center; justify-content: center; transition: background 0.2s;"
            on:click={() => dispatch("openSettings")}
            title="Settings"
            on:mouseover={e => e.currentTarget.style.background='rgba(255,255,255,0.05)'}
            on:mouseout={e => e.currentTarget.style.background='transparent'}
        >
            <i class="fa-solid fa-gear"></i>
        </button>
    </div>
</header>

