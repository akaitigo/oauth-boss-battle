import { useState } from "react";
import { usePKCE } from "../../hooks/usePKCE";
import { useBossProgress } from "../../hooks/useBossProgress";
import type { BossResult } from "../../types/boss";

const API_BASE = "/api/boss/1";

export function Boss1Page() {
	const { pkceState, generate, clear } = usePKCE();
	const { isDefeated, markDefeated } = useBossProgress();
	const [authCode, setAuthCode] = useState<string>("");
	const [result, setResult] = useState<BossResult | null>(null);
	const [phase, setPhase] = useState<
		"intro" | "authorize" | "token" | "result"
	>("intro");

	const defeated = isDefeated(1);

	const handleAuthorizeWithoutPKCE = async () => {
		clear();
		const res = await fetch(`${API_BASE}/authorize`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				client_id: "demo-client",
				redirect_uri: "http://localhost:3000/callback",
			}),
		});
		const data: BossResult = await res.json();
		setResult(data);
		if (data.code) {
			setAuthCode(data.code);
		}
		setPhase("authorize");
	};

	const handleAuthorizeWithPKCE = async () => {
		const pkce = await generate();
		const res = await fetch(`${API_BASE}/authorize`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				client_id: "demo-client",
				redirect_uri: "http://localhost:3000/callback",
				code_challenge: pkce.challenge,
				code_challenge_method: pkce.method,
			}),
		});
		const data: BossResult = await res.json();
		setResult(data);
		if (data.code) {
			setAuthCode(data.code);
		}
		setPhase("authorize");
	};

	const handleTokenExchange = async (withVerifier: boolean) => {
		const body: Record<string, string> = {
			grant_type: "authorization_code",
			code: authCode,
			redirect_uri: "http://localhost:3000/callback",
		};
		if (withVerifier && pkceState) {
			body.code_verifier = pkceState.verifier;
		}

		const res = await fetch(`${API_BASE}/token`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify(body),
		});
		const data: BossResult = await res.json();
		setResult(data);
		if (data.defeated) {
			markDefeated(1);
		}
		setPhase("result");
	};

	const handleReset = () => {
		setPhase("intro");
		setAuthCode("");
		setResult(null);
		clear();
	};

	return (
		<div style={{ maxWidth: 800, margin: "0 auto", padding: 20 }}>
			<h1>Boss 1: PKCE Missing Attack</h1>
			<p>
				Authorization Code Interception Attack — RFC 7636 (PKCE) prevents
				attackers from exchanging stolen authorization codes for tokens.
			</p>

			{defeated && (
				<div
					style={{
						background: "#d4edda",
						border: "1px solid #c3e6cb",
						padding: 16,
						borderRadius: 8,
						marginBottom: 16,
					}}
				>
					Boss Defeated!
				</div>
			)}

			{phase === "intro" && (
				<div>
					<h2>Step 1: Authorization Request</h2>
					<p>Choose how to send the authorization request:</p>
					<div style={{ display: "flex", gap: 16 }}>
						<button
							onClick={handleAuthorizeWithoutPKCE}
							style={{
								padding: "12px 24px",
								background: "#dc3545",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Authorize WITHOUT PKCE (Vulnerable)
						</button>
						<button
							onClick={handleAuthorizeWithPKCE}
							style={{
								padding: "12px 24px",
								background: "#28a745",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Authorize WITH PKCE (Protected)
						</button>
					</div>
				</div>
			)}

			{phase === "authorize" && (
				<div>
					<h2>Step 2: Token Exchange</h2>
					{result && (
						<div
							style={{
								background: "#f8f9fa",
								padding: 16,
								borderRadius: 8,
								marginBottom: 16,
							}}
						>
							<strong>Server Response:</strong>
							<p>{result.message}</p>
							{result.explanation && (
								<p style={{ color: "#666" }}>{result.explanation}</p>
							)}
						</div>
					)}
					<p>
						Authorization Code: <code>{authCode}</code>
					</p>
					{pkceState && (
						<div
							style={{
								background: "#e2e3e5",
								padding: 12,
								borderRadius: 8,
								marginBottom: 16,
							}}
						>
							<p>
								<strong>PKCE State:</strong>
							</p>
							<p>
								code_verifier:{" "}
								<code style={{ wordBreak: "break-all" }}>
									{pkceState.verifier}
								</code>
							</p>
							<p>
								code_challenge:{" "}
								<code style={{ wordBreak: "break-all" }}>
									{pkceState.challenge}
								</code>
							</p>
							<p>
								method: <code>{pkceState.method}</code>
							</p>
						</div>
					)}
					<div style={{ display: "flex", gap: 16 }}>
						<button
							onClick={() => handleTokenExchange(false)}
							style={{
								padding: "12px 24px",
								background: "#dc3545",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Exchange WITHOUT code_verifier (Attacker)
						</button>
						<button
							onClick={() => handleTokenExchange(true)}
							disabled={!pkceState}
							style={{
								padding: "12px 24px",
								background: pkceState ? "#28a745" : "#6c757d",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Exchange WITH code_verifier (Defender)
						</button>
					</div>
				</div>
			)}

			{phase === "result" && result && (
				<div>
					<h2>Result</h2>
					<div
						style={{
							background: result.defeated ? "#d4edda" : "#f8d7da",
							border: `1px solid ${result.defeated ? "#c3e6cb" : "#f5c6cb"}`,
							padding: 16,
							borderRadius: 8,
							marginBottom: 16,
						}}
					>
						<strong>
							{result.defeated ? "BOSS DEFEATED!" : "Attack Succeeded..."}
						</strong>
						<p>{result.message}</p>
						{result.explanation && (
							<p style={{ color: "#666" }}>{result.explanation}</p>
						)}
						{result.rfc_link && (
							<p>
								<a
									href={result.rfc_link}
									target="_blank"
									rel="noopener noreferrer"
								>
									RFC 7636 - PKCE Specification
								</a>
							</p>
						)}
					</div>
					<button
						onClick={handleReset}
						style={{
							padding: "12px 24px",
							background: "#007bff",
							color: "white",
							border: "none",
							borderRadius: 8,
							cursor: "pointer",
						}}
					>
						Try Again
					</button>
				</div>
			)}
		</div>
	);
}
