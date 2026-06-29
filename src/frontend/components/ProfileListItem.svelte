<script>
import { api } from "../lib/api.js";
import { showToast } from "../stores/store.js";
export let profile;
export let onStart;
export let onStop;
export let onDelete;
export let selected = false;
export let onMouseDown = () => {};
export let onHoverSelect = () => {};

let showMenu = false;
let menuStyle = "";

function toggleMenu(e) {
	e.stopPropagation();
	if (!showMenu) {
		const rect = e.currentTarget.getBoundingClientRect();
		menuStyle = `top: ${rect.bottom + 6}px; right: ${window.innerWidth - rect.right}px;`;
	}
	showMenu = !showMenu;
}

function closeMenu() {
	showMenu = false;
}

function handleStart(e) {
	e.stopPropagation();
	onStart(profile.id);
}

function handleStop(e) {
	e.stopPropagation();
	if (onStop) onStop(profile.id);
}

function handleDelete() {
	closeMenu();
	if (profile.status === "running") {
		alert("Không thể xoá profile đang chạy. Hãy dừng lại trước!");
		return;
	}
	if (
		confirm(
			`Bạn có chắc muốn xoá profile "${profile.name}"? Hành động này không thể hoàn tác!`,
		)
	) {
		onDelete(profile.id);
	}
}

async function handleBackup() {
	closeMenu();
	if (!profile || profile.status === "running") {
		showToast("Chỉ có thể backup khi profile đang tắt!", "error");
		return;
	}
	try {
		await api.backupProfile(profile.id);
		showToast(`Backup thành công: ${profile.id}.zip`, "success");
	} catch (err) {
		console.error(err);
		showToast(`Lỗi khi backup: ${err.message}`, "error");
	}
}

$: isRunning = profile.status === "running";
$: isStarting = profile.starting === true;
</script>

<svelte:window on:mousedown={closeMenu} />

<div
  role="button"
  tabindex="0"
  class="list-item selectable-item {selected ? 'selected' : ''} {isRunning ? 'item-running' : ''}"
  data-id={profile.id}
  on:mousedown={onMouseDown}
  on:mouseenter={onHoverSelect}
>
  <!-- Checkbox -->
  <div class="item-checkbox">
    <input type="checkbox" checked={selected} on:mousedown|stopPropagation={onMouseDown} />
  </div>

  <!-- Running indicator bar -->
  {#if isRunning}
    <div class="running-bar"></div>
  {/if}

  <!-- Profile info -->
  <div class="item-info">
    <div class="item-name">{profile.name}</div>
  </div>

  <!-- Status -->
  <div class="item-status">
    {#if profile.backingUp}
      <span class="status-badge status-backup">
        <i class="fa-solid fa-cloud-arrow-up fa-fade" style="font-size: 0.65rem;"></i>
        Backing up...
      </span>
    {/if}

    {#if isRunning && profile.port}
      <span class="port-info">:{profile.port}</span>
    {/if}
  </div>

  <!-- Actions -->
  <div class="item-actions" on:mousedown|stopPropagation>
    <!-- Start / Stop button -->
    {#if isRunning}
      <button
        class="btn btn-action btn-stop"
        on:mousedown={handleStop}
        title="Stop"
        disabled={profile.backingUp}
      >
        <i class="fa-solid fa-stop"></i>
        <span>Stop</span>
      </button>
    {:else}
      <button
        class="btn btn-action btn-start"
        on:mousedown={handleStart}
        title="Start"
        disabled={isStarting}
      >
        {#if isStarting}
          <i class="fa-solid fa-spinner fa-spin"></i>
          <span>Starting...</span>
        {:else}
          <i class="fa-solid fa-play"></i>
          <span>Start</span>
        {/if}
      </button>
    {/if}

    <!-- Gear menu -->
    <div class="menu-container">
      <button class="btn btn-icon-sm" on:mousedown={toggleMenu} title="More options">
        <i class="fa-solid fa-ellipsis-vertical"></i>
      </button>
    </div>
  </div>
</div>

{#if showMenu}
  <div class="dropdown-menu" style={menuStyle} on:mousedown|stopPropagation>
    <button class="menu-item" on:mousedown={handleBackup} disabled={profile.status === "running"}>
      <i class="fa-solid fa-cloud-arrow-up"></i>
      Backup ZIP
    </button>
    <div class="menu-divider"></div>
    <button class="menu-item menu-item-danger" on:mousedown={handleDelete} disabled={profile.status === "running"}>
      <i class="fa-solid fa-trash"></i>
      Xoá Profile
    </button>
  </div>
{/if}

<style>
  .list-item {
    display: flex;
    align-items: center;
    padding: 10px 14px;
    background: var(--card-bg, rgba(30, 41, 59, 0.5));
    border-bottom: 1px solid rgba(255, 255, 255, 0.04);
    cursor: pointer;
    transition: background 0.15s, box-shadow 0.15s;
    position: relative;
    gap: 0;
  }
  
  .list-item:hover {
    background: rgba(255, 255, 255, 0.03);
  }

  .list-item.selected {
    background: rgba(99, 102, 241, 0.08);
  }

  .list-item.item-running {
    background: rgba(16, 185, 129, 0.04);
  }

  .list-item.item-running:hover {
    background: rgba(16, 185, 129, 0.07);
  }

  /* Green left bar for running profiles */
  .running-bar {
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 3px;
    background: var(--success-color, #10b981);
    border-radius: 0 2px 2px 0;
    box-shadow: 0 0 8px rgba(16, 185, 129, 0.4);
  }

  .item-checkbox {
    margin-right: 14px;
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }

  .item-checkbox input {
    width: 15px;
    height: 15px;
    cursor: pointer;
    accent-color: #6366f1;
  }

  .item-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .item-name {
    font-weight: 600;
    font-size: 0.875rem;
    color: var(--text-color);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .item-folder {
    font-size: 0.72rem;
    color: var(--text-muted);
    display: flex;
    align-items: center;
    gap: 4px;
  }

  .item-status {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
    margin-right: 14px;
  }

  .port-info {
    font-size: 0.72rem;
    color: var(--text-muted);
    font-family: monospace;
  }

  /* Status badges */
  .status-badge {
    padding: 3px 8px;
    border-radius: 9999px;
    font-size: 0.72rem;
    font-weight: 600;
    display: inline-flex;
    align-items: center;
    gap: 5px;
    white-space: nowrap;
  }

  .status-stopped {
    background: rgba(148, 163, 184, 0.08);
    color: var(--text-muted);
    border: 1px solid rgba(148, 163, 184, 0.15);
  }

  .status-running {
    background: rgba(16, 185, 129, 0.12);
    color: #10b981;
    border: 1px solid rgba(16, 185, 129, 0.25);
  }

  .status-starting {
    background: rgba(245, 158, 11, 0.12);
    color: #f59e0b;
    border: 1px solid rgba(245, 158, 11, 0.25);
  }

  .status-backup {
    background: rgba(245, 158, 11, 0.12);
    color: #ffc107;
    border: 1px solid rgba(245, 158, 11, 0.2);
  }

  /* Pulse dot */
  .status-dot-pulse {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #10b981;
    box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.4);
    animation: pulse 1.8s infinite;
    flex-shrink: 0;
  }

  @keyframes pulse {
    0%   { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.5); }
    70%  { box-shadow: 0 0 0 5px rgba(16, 185, 129, 0); }
    100% { box-shadow: 0 0 0 0 rgba(16, 185, 129, 0); }
  }

  /* Action buttons */
  .item-actions {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .btn-action {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    padding: 5px 12px;
    border-radius: 6px;
    font-size: 0.78rem;
    font-weight: 600;
    font-family: inherit;
    border: 1px solid transparent;
    cursor: pointer;
    transition: all 0.15s;
    white-space: nowrap;
  }

  .btn-action:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .btn-start {
    background: rgba(99, 102, 241, 0.15);
    color: #818cf8;
    border-color: rgba(99, 102, 241, 0.3);
  }

  .btn-start:not(:disabled):hover {
    background: rgba(99, 102, 241, 0.25);
    border-color: rgba(99, 102, 241, 0.5);
    color: #a5b4fc;
  }

  .btn-stop {
    background: rgba(239, 68, 68, 0.1);
    color: #f87171;
    border-color: rgba(239, 68, 68, 0.25);
  }

  .btn-stop:not(:disabled):hover {
    background: rgba(239, 68, 68, 0.2);
    border-color: rgba(239, 68, 68, 0.4);
  }

  .btn-icon-sm {
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid rgba(255, 255, 255, 0.08);
    color: var(--text-muted);
    border-radius: 6px;
    padding: 5px 8px;
    cursor: pointer;
    font-size: 0.8rem;
    transition: all 0.15s;
    font-family: inherit;
  }

  .btn-icon-sm:hover {
    background: rgba(255, 255, 255, 0.1);
    color: var(--text-color);
  }

  /* Context menu */
  .menu-container {
    position: relative;
  }

  .dropdown-menu {
    position: fixed;
    background: #1a1a2e;
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 8px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.6);
    min-width: 160px;
    z-index: 99999;
    overflow: hidden;
    animation: fadeIn 0.1s ease;
  }

  @keyframes fadeIn {
    from { opacity: 0; transform: translateY(-4px); }
    to   { opacity: 1; transform: translateY(0); }
  }

  .menu-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 9px 14px;
    background: none;
    border: none;
    color: var(--text-color);
    font-size: 0.85rem;
    cursor: pointer;
    transition: background 0.12s;
    text-align: left;
    font-family: inherit;
  }

  .menu-item:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.07);
  }

  .menu-item:disabled {
    opacity: 0.35;
    cursor: not-allowed;
  }

  .menu-item i {
    width: 16px;
    text-align: center;
    color: var(--text-muted);
    font-size: 0.8rem;
  }

  .menu-item-danger { color: #ef4444; }
  .menu-item-danger i { color: #ef4444; }
  .menu-item-danger:hover:not(:disabled) { background: rgba(239, 68, 68, 0.1); }

  .menu-divider {
    height: 1px;
    background: rgba(255, 255, 255, 0.07);
    margin: 2px 0;
  }
</style>
