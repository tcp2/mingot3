# Grid Layout System Redesign

The frontend currently attempts to calculate grid layout based on the number of selected items at once. However, this approach fails when profiles are started sequentially (e.g., starting 1, then later starting another 1) because the frontend doesn't know about previously opened windows. 

To solve this, we will move the grid allocation logic to the Backend.

## Proposed Changes

### 1. Global Slot Manager (Backend)
- Create a global slot manager to track running profile IDs.
- When a profile starts, the backend finds the lowest available empty slot (0, 1, 2, ...).
- When a profile stops (either via Stop button or closing manually), the backend frees that slot.

### 2. Grid Calculation (Backend)
- In `routes/profiles.ts` (`/start/:id`), calculate `(x, y, w, h)` based on the assigned `slotIndex`.
- The Frontend will just pass `screenWidth` and `screenHeight` so the Backend knows the bounds.
- **Layout strategy**: Default to 2 columns and 2 rows (4 slots per screen). If more than 4 profiles are opened, it wraps around or overlaps slightly.
  - `w = screenWidth / 2`
  - `h = screenHeight / 2`
  - `x = (slotIndex % 2) * w`
  - `y = (Math.floor(slotIndex / 2) % 2) * h`

### 3. Deallocation
- Update the `/stop/:id` API to free the slot.
- Update `orbita.ts` -> `proc.on("exit")` event to also free the slot, ensuring slots are cleared even if the user manually clicks the 'X' button on the browser.

### 4. Frontend cleanup
- Remove `gridConfig` calculation from `App.svelte`.
- Pass `screenWidth: window.screen.availWidth` and `screenHeight: window.screen.availHeight` in the `/start/:id` payload.

## User Review Required

- Is a 2x2 grid (4 windows on screen at once) a good default? Or do you prefer a different layout, such as dividing into columns only (e.g., 2 side-by-side taking full height)?
