<script>
export let toasts = [];
</script>

<div class="toast-container">
    {#each toasts as toast (toast.id)}
        <div class="toast toast-{toast.type}">
            {#if toast.type === 'success'}
                <i class="fa-solid fa-check-circle" style="color: var(--success-color)"></i>
            {:else}
                <i class="fa-solid fa-circle-exclamation" style="color: var(--danger-color)"></i>
            {/if}
            <span>{toast.message}</span>
        </div>
    {/each}
</div>
