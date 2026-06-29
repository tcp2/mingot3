import { writable, derived } from "svelte/store";

// -----------------------------------------
// 1. UI & Global State
// -----------------------------------------
export const loading = writable(true);
export const creating = writable(false);
export const toasts = writable([]);
export const searchQuery = writable("");

export function showToast(message, type = "success") {
	const id = Date.now() + Math.random();
	toasts.update((t) => [...t, { id, message, type }]);
	setTimeout(() => {
		toasts.update((t) => t.filter((x) => x.id !== id));
	}, 4000);
}

// -----------------------------------------
// 2. Profiles State
// -----------------------------------------
export const profilesData = writable([]);

export const filteredProfiles = derived(
	[profilesData, searchQuery],
	([$profilesData, $searchQuery]) => {
		return $profilesData.filter(
			(p) =>
				p.name.toLowerCase().includes($searchQuery.toLowerCase()) ||
				p.id.toLowerCase().includes($searchQuery.toLowerCase()),
		);
	},
);

// -----------------------------------------
// 3. Pagination State
// -----------------------------------------
export const currentPage = writable(1);
export const itemsPerPage = writable(50);

export const totalPages = derived(
	[filteredProfiles, itemsPerPage],
	([$filteredProfiles, $itemsPerPage]) => {
		return Math.max(1, Math.ceil($filteredProfiles.length / $itemsPerPage));
	},
);

export const paginatedProfiles = derived(
	[filteredProfiles, currentPage, itemsPerPage],
	([$filteredProfiles, $currentPage, $itemsPerPage]) => {
		// Automatically adjust currentPage if it's out of bounds
		const maxPage = Math.max(
			1,
			Math.ceil($filteredProfiles.length / $itemsPerPage),
		);
		const safePage = Math.min($currentPage, maxPage);
		if (safePage !== $currentPage && safePage > 0) {
			currentPage.set(safePage);
		}

		const start = (safePage - 1) * $itemsPerPage;
		return $filteredProfiles.slice(start, start + $itemsPerPage);
	},
);

// -----------------------------------------
// 4. Folder State
// -----------------------------------------
export const expandedFolders = writable(new Set(["Uncategorized"]));
export const customFolders = writable([]);
export const folderOrder = writable([]);
export const showMoveDropdown = writable(false);
export const draggedFolder = writable(null);

export const groupedProfiles = derived(
	[paginatedProfiles, customFolders],
	([$paginatedProfiles, $customFolders]) => {
		const acc = {};
		for (const folder of $customFolders) {
			acc[folder] = [];
		}
		for (const profile of $paginatedProfiles) {
			const folder = profile.folder || "Uncategorized";
			if (!acc[folder]) acc[folder] = [];
			acc[folder].push(profile);
		}
		return acc;
	},
);

// Sync missing folders into folderOrder
derived(
	[customFolders, groupedProfiles, folderOrder],
	([$customFolders, $groupedProfiles, $folderOrder]) => {
		let orderChanged = false;
		const newOrder = [...$folderOrder];

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
		return { orderChanged, newOrder };
	},
).subscribe(({ orderChanged, newOrder }) => {
	if (orderChanged) {
		folderOrder.set(newOrder);
	}
});

// -----------------------------------------
// 5. Selection State
// -----------------------------------------
export const selectedProfileIds = writable(new Set());
export const isDraggingSelect = writable(false);
export const dragSelectionMode = writable("select");
export const lastSelectedIndex = writable(-1);

// -----------------------------------------
// 6. LocalStorage Initialization
// -----------------------------------------
if (typeof window !== "undefined" && window.localStorage) {
	try {
		const storedExpanded = localStorage.getItem("mingot_expandedFolders");
		if (storedExpanded)
			expandedFolders.set(new Set(JSON.parse(storedExpanded)));

		const storedCustom = localStorage.getItem("mingot_customFolders");
		if (storedCustom) customFolders.set(JSON.parse(storedCustom));

		const storedItemsPerPage = localStorage.getItem("mingot_itemsPerPage");
		if (storedItemsPerPage) itemsPerPage.set(parseInt(storedItemsPerPage, 10));

		const storedFolderOrder = localStorage.getItem("mingot_folderOrder");
		if (storedFolderOrder) folderOrder.set(JSON.parse(storedFolderOrder));
	} catch (e) {
		console.warn("Failed to load state from localStorage");
	}

	expandedFolders.subscribe((v) =>
		localStorage.setItem(
			"mingot_expandedFolders",
			JSON.stringify(Array.from(v)),
		),
	);
	customFolders.subscribe((v) =>
		localStorage.setItem("mingot_customFolders", JSON.stringify(v)),
	);
	itemsPerPage.subscribe((v) =>
		localStorage.setItem("mingot_itemsPerPage", v.toString()),
	);
	folderOrder.subscribe((v) => {
		if (v.length > 0)
			localStorage.setItem("mingot_folderOrder", JSON.stringify(v));
	});
}
