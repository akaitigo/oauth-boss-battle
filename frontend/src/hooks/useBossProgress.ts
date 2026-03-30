import { useState, useCallback } from "react";

const STORAGE_KEY = "oauth-boss-battle-progress";

interface BossProgress {
	[bossId: number]: boolean;
}

function loadProgress(): BossProgress {
	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored) {
			return JSON.parse(stored) as BossProgress;
		}
	} catch {
		// Ignore parse errors
	}
	return {};
}

function saveProgress(progress: BossProgress): void {
	localStorage.setItem(STORAGE_KEY, JSON.stringify(progress));
}

export function useBossProgress() {
	const [progress, setProgress] = useState<BossProgress>(loadProgress);

	const markDefeated = useCallback((bossId: number) => {
		setProgress((prev) => {
			const next = { ...prev, [bossId]: true };
			saveProgress(next);
			return next;
		});
	}, []);

	const isDefeated = useCallback(
		(bossId: number): boolean => {
			return progress[bossId] === true;
		},
		[progress],
	);

	const resetProgress = useCallback(() => {
		setProgress({});
		localStorage.removeItem(STORAGE_KEY);
	}, []);

	return { progress, markDefeated, isDefeated, resetProgress };
}
