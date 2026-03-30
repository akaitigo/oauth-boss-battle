import { useState, useCallback } from "react";

function base64UrlEncode(buffer: ArrayBuffer): string {
	const bytes = new Uint8Array(buffer);
	let binary = "";
	for (const b of bytes) {
		binary += String.fromCharCode(b);
	}
	return btoa(binary)
		.replace(/\+/g, "-")
		.replace(/\//g, "_")
		.replace(/=+$/, "");
}

async function generateVerifier(): Promise<string> {
	const buffer = new Uint8Array(32);
	crypto.getRandomValues(buffer);
	return base64UrlEncode(buffer.buffer as ArrayBuffer);
}

async function computeS256Challenge(verifier: string): Promise<string> {
	const encoder = new TextEncoder();
	const data = encoder.encode(verifier);
	const hash = await crypto.subtle.digest("SHA-256", data);
	return base64UrlEncode(hash);
}

interface PKCEState {
	verifier: string;
	challenge: string;
	method: string;
}

export function usePKCE() {
	const [pkceState, setPKCEState] = useState<PKCEState | null>(null);

	const generate = useCallback(async () => {
		const verifier = await generateVerifier();
		const challenge = await computeS256Challenge(verifier);
		const state: PKCEState = { verifier, challenge, method: "S256" };
		setPKCEState(state);
		return state;
	}, []);

	const clear = useCallback(() => {
		setPKCEState(null);
	}, []);

	return { pkceState, generate, clear };
}
