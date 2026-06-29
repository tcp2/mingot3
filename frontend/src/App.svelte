<script>
import { onMount } from "svelte";
import "./app.css";
import Header from "./components/Header.svelte";
import SettingsModal from "./components/SettingsModal.svelte";
import ProfileListItem from "./components/ProfileListItem.svelte";
import Toast from "./components/Toast.svelte";
import { api, onProfileEvent } from "./lib/api.js";

let showSettingsModal = false;


import {
	loading,
	creating,
	toasts,
	searchQuery,
	profilesData,
	filteredProfiles,
	currentPage,
	itemsPerPage,
	totalPages,
	paginatedProfiles,
	expandedFolders,
	customFolders,
	folderOrder,
	showMoveDropdown,
	draggedFolder,
	groupedProfiles,
	selectedProfileIds,
	isDraggingSelect,
	dragSelectionMode,
	lastSelectedIndex,
	showToast,
} from "./stores/store.js";

$: selectedProfiles = $profilesData.filter((p) =>
	$selectedProfileIds.has(p.id),
);
$: startableCount = selectedProfiles.filter(
	(p) => p.status !== "running",
).length;
$: stoppableCount = selectedProfiles.filter(
	(p) => p.status === "running",
).length;

$: {
	// Sync folderOrder with customFolders and groupedProfiles
	let orderChanged = false;
	let newOrder = [...$folderOrder];

	const allKnownFolders = new Set([
		...$customFolders,
		...Object.keys($groupedProfiles),
	]);
	for (const f of allKnownFolders) {
		if (!newOrder.includes(f)) {
			newOrder.push(f);
			orderChanged = true;
		}
	}
	if (orderChanged) $folderOrder = newOrder;
}

function handleDragStart(e, folder) {
	$draggedFolder = folder;
	e.dataTransfer.effectAllowed = "move";

	const headerElement = e.target.closest(".folder-header");
	if (headerElement) {
		e.dataTransfer.setDragImage(headerElement, 20, 20);
	}
}

function handleDragOver(e) {
	e.preventDefault();
	e.dataTransfer.dropEffect = "move";
}

function handleDrop(e, targetFolder) {
	e.preventDefault();
	if ($draggedFolder && $draggedFolder !== targetFolder) {
		const newOrder = [...$folderOrder];
		const fromIndex = newOrder.indexOf($draggedFolder);
		const toIndex = newOrder.indexOf(targetFolder);

		if (fromIndex !== -1 && toIndex !== -1) {
			newOrder.splice(fromIndex, 1);
			newOrder.splice(toIndex, 0, $draggedFolder);
			$folderOrder = newOrder;
		}
	}
	$draggedFolder = null;
}

function toggleFolder(folder) {
	if ($expandedFolders.has(folder)) {
		$expandedFolders.delete(folder);
	} else {
		$expandedFolders.add(folder);
	}
	$expandedFolders = new Set($expandedFolders);
}

function handleMouseDownItem(id, index, event) {
	if (event.button !== 0) return; // left click only
	if (event.target.closest("button") || event.target.closest("input")) return;

	if (event.shiftKey && $lastSelectedIndex !== -1) {
		const start = Math.min($lastSelectedIndex, index);
		const end = Math.max($lastSelectedIndex, index);
		for (let i = start; i <= end; i++) {
			$selectedProfileIds.add($filteredProfiles[i].id);
		}
		$selectedProfileIds = new Set($selectedProfileIds);
	} else {
		$isDraggingSelect = true;

		const isCheckbox =
			event.target?.tagName &&
			event.target.tagName.toLowerCase() === "input" &&
			event.target.type === "checkbox";

		if (!event.ctrlKey && !event.metaKey && !isCheckbox) {
			if ($selectedProfileIds.size === 1 && $selectedProfileIds.has(id)) {
				$selectedProfileIds.clear();
				$dragSelectionMode = "deselect";
			} else {
				$selectedProfileIds.clear();
				$dragSelectionMode = "select";
				$selectedProfileIds.add(id);
			}
		} else {
			if ($selectedProfileIds.has(id)) {
				$dragSelectionMode = "deselect";
				$selectedProfileIds.delete(id);
			} else {
				$dragSelectionMode = "select";
				$selectedProfileIds.add(id);
			}
		}
		$selectedProfileIds = new Set($selectedProfileIds);
		$lastSelectedIndex = index;
	}
}

function handleHoverSelect(id, index) {
	if (!$isDraggingSelect) return;

	if ($dragSelectionMode === "select") {
		$selectedProfileIds.add(id);
	} else {
		$selectedProfileIds.delete(id);
	}
	$selectedProfileIds = new Set($selectedProfileIds);
	$lastSelectedIndex = index;
}

async function startSelected() {
	// Sắp xếp các id theo thứ tự xuất hiện trong danh sách để mở theo đúng thứ tự từ trên xuống dưới
	const ids = Array.from($selectedProfileIds).sort((a, b) => {
		const indexA = $profilesData.findIndex((p) => p.id === a);
		const indexB = $profilesData.findIndex((p) => p.id === b);
		return indexA - indexB;
	});

	if (ids.length === 0) return;

	for (let i = 0; i < ids.length; i++) {
		startProfile(ids[i]);
		await new Promise((r) => setTimeout(r, 800)); // stagger starts slightly
	}
}

async function stopSelected() {
	const ids = Array.from($selectedProfileIds);
	for (const id of ids) {
		stopProfile(id);
	}
}

async function arrangeWindows() {
	try {
		showToast("Arranging windows...", "success");
		await api.arrangeWindows();
	} catch (e) {
		showToast(e.message, "error");
	}
}

async function executeMove(targetFolder) {
	let folderName = targetFolder;
	if (targetFolder === "__NEW__") {
		folderName = window.prompt("Enter new folder name:");
		if (!folderName || folderName.trim() === "") return;
		folderName = folderName.trim();
	}

	const ids = Array.from($selectedProfileIds);
	try {
		await api.updateFolder(ids, folderName);
		showToast(`Moved ${ids.length} profiles to '${folderName}'`, "success");
		$selectedProfileIds = new Set();
		$showMoveDropdown = false;
		if (folderName !== "Uncategorized") {
			$expandedFolders.add(folderName);
			$expandedFolders = new Set($expandedFolders);
		}
		await fetchProfiles();
	} catch (err) {
		showToast(`Error moving profiles: ${err.message}`, "error");
	}
}

function createNewFolder() {
	const folderName = window.prompt("Enter new folder name:");
	if (folderName && folderName.trim() !== "") {
		const name = folderName.trim();
		if (!$customFolders.includes(name)) {
			$customFolders = [...$customFolders, name];
			$expandedFolders.add(name);
			$expandedFolders = new Set($expandedFolders);
		}
	}
}

onMount(() => {
	fetchProfiles();

	// ── Real-time push events (Electron only) ────────────────────────────────
	// When a browser closes externally, orbita.ts emits events that are
	// forwarded here via IPC push — no polling delay.
	const unsubscribe = onProfileEvent((event) => {
		const { type, id } = event;
		const index = $profilesData.findIndex((p) => p.id === id);
		if (index === -1) return;

		if (type === "profile:closed") {
			$profilesData[index] = {
				...$profilesData[index],
				status: "stopped",
				port: null,
				starting: false,
			};
			$profilesData = [...$profilesData];
		} else if (type === "profile:backing-up") {
			$profilesData[index] = { ...$profilesData[index], backingUp: true };
			$profilesData = [...$profilesData];
		} else if (type === "profile:backed-up") {
			$profilesData[index] = { ...$profilesData[index], backingUp: false };
			$profilesData = [...$profilesData];
		}
	});

	// Poll every 3s — but only when there are running profiles
	const interval = setInterval(() => {
		// With IPC push events, we don't need to poll anymore
		// const hasRunning = $profilesData.some(p => p.status === 'running');
		// if (hasRunning) fetchProfiles();
	}, 3000);

	const handleGlobalMouseUp = () => {
		$isDraggingSelect = false;
	};
	const handleGlobalClick = (e) => {
		if ($showMoveDropdown && !e.target.closest(".move-dropdown-container")) {
			$showMoveDropdown = false;
		}
	};

	window.addEventListener("mouseup", handleGlobalMouseUp);
	window.addEventListener("click", handleGlobalClick);

	return () => {
		window.removeEventListener("mouseup", handleGlobalMouseUp);
		window.removeEventListener("click", handleGlobalClick);
		clearInterval(interval);
		unsubscribe();
	};
});

async function fetchProfiles() {
	$loading = true;
	try {
		const data = await api.listProfiles();
		$profilesData = data.profiles;
	} catch (_err) {
		showToast("Failed to connect to server", "error");
	} finally {
		$loading = false;
	}
}

async function createProfile() {
	$creating = true;
	try {
		await api.createProfile();
		showToast("Profile created successfully!", "success");
		await fetchProfiles();
	} catch (err) {
		showToast(`Error: ${err.message}`, "error");
	} finally {
		$creating = false;
	}
}

let highlightedProfileId = null;

let importing = false;
async function handleImport(e) {
	const file = e.target.files[0];
	if (!file) return;

	importing = true;
	try {
		await api.importProfile(file);
		showToast("Profile imported successfully!", "success");
		await fetchProfiles();
	} catch (err) {
		if (err.code === "CONFLICT" && err.profileId) {
			highlightedProfileId = err.profileId;
			setTimeout(() => (highlightedProfileId = null), 7000);

			if (
				confirm(
					`Profile "${err.profileId}" đã tồn tại.\nBạn có muốn xoá cái cũ và đè lên không?`,
				)
			) {
				try {
					importing = true;
					await api.importProfile(file, true);
					showToast("Đã đè profile thành công!", "success");
					await fetchProfiles();
				} catch (e) {
					showToast(`Lỗi khi đè profile: ${e.message}`, "error");
				}
			} else {
				showToast("Đã huỷ import", "error");
			}
		} else {
			showToast(`Lỗi: ${err.message}`, "error");
		}
	} finally {
		importing = false;
		e.target.value = "";
	}
}

async function startProfile(id) {
	const index = $profilesData.findIndex((p) => p.id === id);
	if (index === -1) return;

	$profilesData[index].starting = true;
	$profilesData = [...$profilesData];

	showToast("Starting browser...", "success");
	try {
		const data = await api.startProfile(id);
		showToast(`Browser started on port ${data.port}`, "success");
		$profilesData[index].status = "running";
		$profilesData[index].port = data.port;
	} catch (err) {
		showToast(`Failed to start: ${err.message}`, "error");
	} finally {
		$profilesData[index].starting = false;
		$profilesData = [...$profilesData];
	}
}

async function stopProfile(id) {
	const index = $profilesData.findIndex((p) => p.id === id);
	if (index === -1) return;

	// Optimistic UI: đánh dấu stopped ngay — badge "Backing up" sẽ đến
	// qua push event profile:backing-up từ proc.on("exit") trong orbita.ts.
	// Không set backingUp ở đây để tránh conflict với push events.
	$profilesData[index].status = "stopped";
	$profilesData[index].port = null;
	$profilesData = [...$profilesData];

	showToast("Stopping...", "success");
	try {
		await api.stopProfile(id);
	} catch (err) {
		// NOT_FOUND is fine — browser already closed
		if (err?.code !== "NOT_FOUND") {
			showToast(`Failed to stop: ${err.message}`, "error");
		}
	}
	// backingUp lifecycle is fully managed by push events:
	//   profile:backing-up → backingUp = true
	//   profile:backed-up  → backingUp = false
	// No setTimeout needed.
}

async function deleteProfile(id) {
	try {
		await api.deleteProfile(id);
		showToast("Profile đã được xoá!", "success");
		$profilesData = $profilesData.filter((p) => p.id !== id);
		$selectedProfileIds.delete(id);
		$selectedProfileIds = new Set($selectedProfileIds);
	} catch (err) {
		showToast(`Lỗi khi xoá: ${err.message}`, "error");
	}
}

async function checkAllStatuses() {
	try {
		const data = await api.listProfiles();
		let changed = false;
		const updated = $profilesData.map((p) => {
			const fresh = data.profiles.find((f) => f.id === p.id);
			if (!fresh) return p;

			// Detect a browser that was closed externally (was running, now stopped)
			if (p.status === "running" && fresh.status === "stopped") {
				changed = true;
				return { ...p, status: "stopped", port: null, starting: false };
			}

			// Safety-net: if stopped but backingUp badge is stuck (push event missed in dev mode),
			// clear it after the next poll cycle confirms the profile is truly stopped.
			if (fresh.status === "stopped" && p.backingUp) {
				changed = true;
				return { ...p, backingUp: false };
			}

			// Also sync port in case it changed
			if (p.port !== fresh.port) {
				changed = true;
				return { ...p, port: fresh.port };
			}
			return p;
		});
		if (changed) $profilesData = updated;
	} catch (_e) {}
}
</script>

<div class="app-container">
  <Header on:openSettings={() => showSettingsModal = true} />

  <main style="padding-bottom: 80px;">
    <div class="toolbar">
      <header
        class="controls-bar"
        style="display: flex; flex-wrap: wrap; gap: 10px; justify-content: space-between; align-items: center; padding: 10px 20px; background: rgba(255,255,255,0.03); border-bottom: 1px solid rgba(255,255,255,0.05);"
      >
        <div class="actions" style="display: flex; gap: 10px;">
          <button
            class="btn btn-primary btn-sm"
            style="padding: 5px 10px; font-size: 0.85rem;"
            on:click={createProfile}
            disabled={$creating}
          >
            {#if $creating}
              <i class="fa-solid fa-spinner fa-spin"></i> Creating...
            {:else}
              <i class="fa-solid fa-plus"></i> Profile
            {/if}
          </button>
          
          <input type="file" id="import-file" accept=".zip" style="display: none" on:change={handleImport} />
          <button class="btn btn-sm" style="padding: 5px 10px; font-size: 0.85rem;" on:click={() => document.getElementById('import-file').click()} disabled={importing}>
            {#if importing}
              <i class="fa-solid fa-spinner fa-spin"></i>
            {:else}
              <i class="fa-solid fa-file-import"></i>
            {/if}
             Import
          </button>

          <button class="btn btn-sm" style="padding: 5px 10px; font-size: 0.85rem;" on:click={createNewFolder}>
            <i class="fa-solid fa-folder-plus"></i> Folder
          </button>
          <button class="btn btn-sm" style="padding: 5px 10px; font-size: 0.85rem;" on:click={arrangeWindows}>
            <i class="fa-solid fa-border-all"></i> Arrange
          </button>
        </div>

        <div class="search-bar" style="flex: 1; min-width: 200px;">
          <i class="fa-solid fa-search"></i>
          <input
            type="text"
            bind:value={$searchQuery}
            placeholder="Search profiles..."
          />
        </div>
      </header>
    </div>

    <div
      style="margin-bottom: 10px; display: flex; justify-content: space-between; align-items: center;"
    >
      <div style="color: var(--text-muted); font-size: 0.9rem;">
        Showing {Math.min(
          ($currentPage - 1) * $itemsPerPage + 1,
          $filteredProfiles.length,
        )} - {Math.min($currentPage * $itemsPerPage, $filteredProfiles.length)} of
        {$filteredProfiles.length} profiles
      </div>
    </div>

    {#if $loading}
      <div
        class="loading-state"
        style="display: flex; gap: 10px; justify-content: center; align-items: center; padding: 3rem;"
      >
        <i
          class="fa-solid fa-spinner fa-spin fa-2x"
          style="color: var(--primary-color);"
        ></i>
        <p style="margin: 0; font-size: 1.2rem;">Loading profiles...</p>
      </div>
    {:else}
      <div class="profile-list">
        {#if $filteredProfiles.length === 0}
          <p
            style="color: var(--text-muted); text-align: center; padding: 2rem;"
          >
            No profiles found.
          </p>
        {:else}
          {#each $folderOrder.filter((f) => $groupedProfiles[f]) as folder (folder)}
            {@const profilesInFolder = $groupedProfiles[folder]}
            <div
              class="folder-group"
              style="margin-bottom: 20px; background: rgba(0,0,0,0.2); border-radius: 8px; overflow: hidden; border: 1px solid rgba(255,255,255,0.05);"
              on:dragover={handleDragOver}
              on:drop={(e) => handleDrop(e, folder)}
            >
              <div
                class="folder-header"
                style="display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; background: rgba(255,255,255,0.03); cursor: pointer;"
                on:click={() => toggleFolder(folder)}
              >
                <div
                  style="display: flex; align-items: center; gap: 10px; font-weight: bold; color: var(--text-color);"
                >
                  <div
                    draggable="true"
                    on:dragstart={(e) => handleDragStart(e, folder)}
                    style="cursor: grab; display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; margin-right: 5px;"
                    on:click|stopPropagation
                  >
                    <i
                      class="fa-solid fa-grip-vertical"
                      style="color: var(--text-muted);"
                    ></i>
                  </div>
                  <i
                    class="fa-solid {$expandedFolders.has(folder)
                      ? 'fa-folder-open'
                      : 'fa-folder'}"
                    style="color: #fbbf24;"
                  ></i>
                  {folder}
                  <span
                    style="font-size: 0.8rem; color: var(--text-muted); font-weight: normal;"
                    >({profilesInFolder.length})</span
                  >
                </div>
                <button
                  class="btn"
                  style="padding: 4px 8px; font-size: 0.8rem; background: rgba(255,255,255,0.05); color: var(--text-muted);"
                  on:click|stopPropagation={() => toggleFolder(folder)}
                >
                  {#if $expandedFolders.has(folder)}
                    <i class="fa-solid fa-chevron-up"></i>
                  {:else}
                    <i class="fa-solid fa-chevron-down"></i>
                  {/if}
                </button>
              </div>

              {#if $expandedFolders.has(folder)}
                <div class="folder-content" style="padding: 10px;">
                  {#each profilesInFolder as profile (profile.id)}
                    <ProfileListItem
                      {profile}
                      selected={$selectedProfileIds.has(profile.id)}
                      onStart={startProfile}
                      onStop={stopProfile}
                      onDelete={deleteProfile}
                      onMouseDown={(event) =>
                        handleMouseDownItem(
                          profile.id,
                          $filteredProfiles.findIndex(
                            (p) => p.id === profile.id,
                          ),
                          event,
                        )}
                      onHoverSelect={() =>
                        handleHoverSelect(
                          profile.id,
                          $filteredProfiles.findIndex(
                            (p) => p.id === profile.id,
                          ),
                        )}
                    />
                  {/each}
                </div>
              {/if}
            </div>
          {/each}
        {/if}
      </div>

      {#if $totalPages > 1}
        <div
          class="pagination"
          style="display: flex; justify-content: center; align-items: center; margin-top: 20px; gap: 15px;"
        >
          <button
            class="btn"
            disabled={$currentPage === 1}
            on:click={() => $currentPage--}
          >
            <i class="fa-solid fa-chevron-left"></i> Prev
          </button>
          <span>Page {$currentPage} of {$totalPages}</span>
          <button
            class="btn"
            disabled={$currentPage === $totalPages}
            on:click={() => $currentPage++}
          >
            Next <i class="fa-solid fa-chevron-right"></i>
          </button>

          <select
            bind:value={$itemsPerPage}
            on:change={() => ($currentPage = 1)}
            style="margin-left: 20px; padding: 6px 10px; background: var(--surface-color); color: var(--text-color); border: 1px solid rgba(255,255,255,0.1); border-radius: 6px; cursor: pointer; outline: none;"
          >
            <option value={20}>20 / page</option>
            <option value={50}>50 / page</option>
            <option value={100}>100 / page</option>
            <option value={500}>500 / page</option>
          </select>
        </div>
      {/if}
    {/if}

    {#if $selectedProfileIds.size > 0}
        <div class="bulk-action-bar" style="position: fixed; bottom: 20px; left: 50%; transform: translateX(-50%); background: var(--surface-color); border: 1px solid rgba(255,255,255,0.1); border-radius: 8px; padding: 10px 20px; display: flex; gap: 15px; box-shadow: 0 10px 25px rgba(0,0,0,0.5); z-index: 1000; align-items: center;">
            <div class="move-dropdown-container" style="position: relative;">
                <button class="btn" on:click={() => $showMoveDropdown = !$showMoveDropdown} style="background: rgba(255,255,255,0.05); color: var(--text-color);">
                    <i class="fa-solid fa-folder-tree"></i> Move <i class="fa-solid fa-caret-up" style="margin-left: 5px;"></i>
                </button>
                {#if $showMoveDropdown}
                    <div class="dropdown-menu" style="position: absolute; bottom: 100%; left: 0; margin-bottom: 5px; background: #1e1e1e; border: 1px solid rgba(255,255,255,0.1); border-radius: 8px; box-shadow: 0 5px 15px rgba(0,0,0,0.5); min-width: 180px; z-index: 1001;">
                        <button style="display: block; width: 100%; text-align: left; padding: 10px 16px; background: none; border: none; color: var(--text-color); cursor: pointer; transition: background 0.2s;" on:mouseover={e => e.currentTarget.style.background='rgba(255,255,255,0.1)'} on:mouseout={e => e.currentTarget.style.background='none'} on:click={() => executeMove('Uncategorized')}>
                            <i class="fa-solid fa-box-open" style="margin-right: 8px; color: var(--text-muted); width: 16px; text-align: center;"></i> Uncategorized
                        </button>
                        <div style="height: 1px; background: rgba(255,255,255,0.1); margin: 4px 0;"></div>
                        <div style="max-height: 250px; overflow-y: auto;">
                        {#each Object.keys($groupedProfiles).sort() as folder}
                            {#if folder !== 'Uncategorized'}
                            <button style="display: block; width: 100%; text-align: left; padding: 10px 16px; background: none; border: none; color: var(--text-color); cursor: pointer; transition: background 0.2s;" on:mouseover={e => e.currentTarget.style.background='rgba(255,255,255,0.1)'} on:mouseout={e => e.currentTarget.style.background='none'} on:click={() => executeMove(folder)}>
                                <i class="fa-solid fa-folder" style="margin-right: 8px; color: #fbbf24; width: 16px; text-align: center;"></i> {folder}
                            </button>
                            {/if}
                        {/each}
                        </div>
                        <div style="height: 1px; background: rgba(255,255,255,0.1); margin: 4px 0;"></div>
                        <button style="display: block; width: 100%; text-align: left; padding: 10px 16px; background: none; border: none; color: #10b981; cursor: pointer; font-weight: bold; transition: background 0.2s;" on:mouseover={e => e.currentTarget.style.background='rgba(255,255,255,0.1)'} on:mouseout={e => e.currentTarget.style.background='none'} on:click={() => executeMove('__NEW__')}>
                            <i class="fa-solid fa-plus" style="margin-right: 8px; width: 16px; text-align: center;"></i> New Folder...
                        </button>
                    </div>
                {/if}
            </div>

            <div style="width: 1px; height: 20px; background: rgba(255,255,255,0.2);"></div>

            <button class="btn {stoppableCount > 0 ? 'btn-danger' : 'btn-primary'}" style="min-width: 140px;" on:click={stoppableCount > 0 ? stopSelected : startSelected}>
                {#if stoppableCount > 0}
                    <i class="fa-solid fa-stop"></i> Stop ({stoppableCount})
                {:else}
                    <i class="fa-solid fa-play"></i> Start ({startableCount})
                {/if}
            </button>
        </div>
    {/if}

    <Toast toasts={$toasts} />
  </main>
  
  <SettingsModal isOpen={showSettingsModal} onClose={() => showSettingsModal = false} />
</div>
