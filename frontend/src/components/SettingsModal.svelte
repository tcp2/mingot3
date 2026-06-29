<script>
    import { onMount } from "svelte";
    import { api } from "../lib/api.js";
    import { showToast } from "../stores/store.js";

    export let isOpen = false;
    export let onClose = () => {};

    let tokens = [];
    let loading = false;
    let selectedToken = "";
    let manualToken = "";

    async function scan() {
        loading = true;
        try {
            const data = await api.scanTokens();
            tokens = data || [];
            
            // Tìm token đang active để tự động tích chọn
            const active = tokens.find(t => t.active);
            if (active) {
                selectedToken = active.token;
            }
            if (tokens.length === 0) {
                showToast("No GoLogin tokens found on this computer.", "info");
            }
        } catch (e) {
            showToast("Failed to scan tokens: " + e.message, "error");
        } finally {
            loading = false;
        }
    }

    async function save() {
        let tokenToSave = selectedToken;
        if (manualToken.trim()) {
            tokenToSave = manualToken.trim();
        }

        if (!tokenToSave) {
            showToast("Please select or enter a token first.", "error");
            return;
        }

        try {
            await api.selectToken(tokenToSave);
            showToast("GoLogin token saved successfully!", "success");
            onClose();
        } catch (e) {
            showToast("Failed to save token: " + e.message, "error");
        }
    }

    async function clear() {
        try {
            await api.clearToken();
            selectedToken = "";
            manualToken = "";
            showToast("Cleared selected token. Launching will require setting a token.", "info");
            // Cập nhật lại danh sách quét
            await scan();
        } catch (e) {
            showToast("Failed to clear token: " + e.message, "error");
        }
    }

    // Tự động scan khi mở modal
    $: if (isOpen) {
        scan();
    }
</script>

{#if isOpen}
    <div class="modal-backdrop" on:click={onClose}>
        <div class="modal-content" on:click|stopPropagation style="max-width: 600px; width: 90%; background: var(--surface-color, #1e1e2e); color: var(--text-color, #f8f8f2); border: 1px solid rgba(255,255,255,0.1); border-radius: 12px; padding: 24px; box-shadow: 0 20px 40px rgba(0,0,0,0.5);">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; border-bottom: 1px solid rgba(255,255,255,0.1); padding-bottom: 12px;">
                <h2 style="margin: 0; font-size: 1.35rem; display: flex; align-items: center; gap: 8px;">
                    <i class="fa-solid fa-gear" style="color: var(--primary-color, #8be9fd);"></i>
                    Settings & Tokens
                </h2>
                <button on:click={onClose} style="background: none; border: none; color: var(--text-muted, #a0a0b0); cursor: pointer; font-size: 1.2rem;">
                    <i class="fa-solid fa-xmark"></i>
                </button>
            </div>

            <div style="margin-bottom: 24px;">
                <h3 style="margin: 0 0 10px 0; font-size: 1rem; color: #f1fa8c;">Add / Paste Token Manually</h3>
                <div style="display: flex; gap: 10px;">
                    <input 
                        type="text" 
                        bind:value={manualToken} 
                        placeholder="Paste your GoLogin JWT Token here..."
                        style="flex: 1; padding: 10px 14px; background: rgba(0,0,0,0.25); border: 1px solid rgba(255,255,255,0.15); border-radius: 6px; color: #fff; font-family: monospace; font-size: 0.85rem;"
                    />
                </div>
            </div>

            <div style="margin-bottom: 24px;">
                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;">
                    <h3 style="margin: 0; font-size: 1rem; color: #50fa7b;">Scanned Tokens on Computer</h3>
                    <button class="btn btn-sm" on:click={scan} disabled={loading} style="padding: 4px 8px; font-size: 0.8rem; background: rgba(255,255,255,0.08);">
                        <i class="fa-solid fa-arrows-rotate {loading ? 'fa-spin' : ''}"></i> Scan
                    </button>
                </div>

                {#if loading}
                    <div style="text-align: center; padding: 20px; color: var(--text-muted);">
                        <i class="fa-solid fa-spinner fa-spin"></i> Scanning local database...
                    </div>
                {:else if tokens.length === 0}
                    <div style="text-align: center; padding: 20px; color: var(--text-muted); background: rgba(0,0,0,0.1); border-radius: 6px; border: 1px dashed rgba(255,255,255,0.1);">
                        No tokens found in GoLogin desktop directories.
                    </div>
                {:else}
                    <div class="token-list-container" style="max-height: 200px; overflow-y: auto; background: rgba(0,0,0,0.15); border-radius: 8px; border: 1px solid rgba(255,255,255,0.08);">
                        {#each tokens as t}
                            <label style="display: flex; align-items: center; gap: 12px; padding: 12px 16px; border-bottom: 1px solid rgba(255,255,255,0.05); cursor: pointer; transition: background 0.2s; margin: 0;" class="token-item">
                                <input 
                                    type="radio" 
                                    name="selected-token" 
                                    value={t.token} 
                                    bind:group={selectedToken}
                                    on:change={() => manualToken = ""} 
                                />
                                <div style="flex: 1;">
                                    <div style="font-weight: 500; font-size: 0.9rem; display: flex; align-items: center; gap: 8px;">
                                        User ID: <span style="font-family: monospace; color: #ff79c6;">{t.userId || 'Unknown'}</span>
                                        {#if t.active}
                                            <span style="font-size: 0.75rem; background: rgba(80, 250, 123, 0.2); color: #50fa7b; padding: 2px 6px; border-radius: 4px; font-weight: bold;">Active</span>
                                        {/if}
                                    </div>
                                    <div style="font-size: 0.75rem; color: var(--text-muted); font-family: monospace; margin-top: 4px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 450px;">
                                        {t.token}
                                    </div>
                                </div>
                            </label>
                        {/each}
                    </div>
                {/if}
            </div>

            <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 24px; border-top: 1px solid rgba(255,255,255,0.1); padding-top: 16px;">
                <button class="btn btn-danger btn-sm" on:click={clear} style="padding: 6px 12px;">
                    <i class="fa-solid fa-trash-can"></i> Clear Config
                </button>
                <div style="display: flex; gap: 10px;">
                    <button class="btn btn-secondary" on:click={onClose}>
                        Cancel
                    </button>
                    <button class="btn btn-primary" on:click={save}>
                        Save Settings
                    </button>
                </div>
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        width: 100vw;
        height: 100vh;
        background: rgba(0, 0, 0, 0.6);
        backdrop-filter: blur(4px);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 9999;
    }
    .token-item:hover {
        background: rgba(255, 255, 255, 0.04);
    }
    .token-list-container::-webkit-scrollbar {
        width: 6px;
    }
    .token-list-container::-webkit-scrollbar-track {
        background: transparent;
    }
    .token-list-container::-webkit-scrollbar-thumb {
        background: rgba(255, 255, 255, 0.15);
        border-radius: 3px;
    }
</style>
