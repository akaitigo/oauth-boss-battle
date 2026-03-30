import { useState } from "react";
import { useBossProgress } from "../../hooks/useBossProgress";
import type { BossResult } from "../../types/boss";

const API_BASE = "/api/boss/4";

type Phase = "intro" | "rotated" | "verify" | "result";

export function Boss4Page() {
	const { isDefeated, markDefeated } = useBossProgress();
	const [phase, setPhase] = useState<Phase>("intro");
	const [token, setToken] = useState("");
	const [result, setResult] = useState<BossResult | null>(null);
	const [cacheMode, setCacheMode] = useState<"stale" | "smart">("stale");
	const [rotationInfo, setRotationInfo] = useState<string>("");

	const defeated = isDefeated(4);

	const handleRotateAndSign = async (revokeOld: boolean) => {
		// Rotate
		const rotateRes = await fetch(`${API_BASE}/rotate`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ revoke_old_key: revokeOld }),
		});
		const rotateData: Record<string, unknown> = await rotateRes.json();
		setRotationInfo(rotateData.message as string);

		// Sign with new key
		const signRes = await fetch(`${API_BASE}/sign`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ payload: '{"sub":"demo-user"}' }),
		});
		const signData: Record<string, unknown> = await signRes.json();
		if (signData.token) {
			setToken(signData.token as string);
		}

		setPhase("rotated");
	};

	const handleConfigureCache = async (smart: boolean) => {
		await fetch(`${API_BASE}/configure-cache`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ smart_mode: smart }),
		});
		setCacheMode(smart ? "smart" : "stale");
	};

	const handleVerify = async () => {
		const res = await fetch(`${API_BASE}/verify`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ token }),
		});
		const data: BossResult = await res.json();
		setResult(data);
		if (data.defeated) markDefeated(4);
		setPhase("result");
	};

	const handleReset = () => {
		setPhase("intro");
		setToken("");
		setResult(null);
		setRotationInfo("");
	};

	return (
		<div style={{ maxWidth: 800, margin: "0 auto", padding: 20 }}>
			<h1>Boss 4: JWKS Rotation Failure</h1>
			<p>
				Key rotation causes token verification failures when JWKS caching is
				misconfigured — RFC 7517 (JWK) and RFC 7515 (JWS).
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
					<h2>Step 1: Configure Cache Strategy</h2>
					<p>
						Current cache mode:{" "}
						<strong>
							{cacheMode === "smart"
								? "Smart (kid-based refresh)"
								: "Stale (TTL-only)"}
						</strong>
					</p>
					<div style={{ display: "flex", gap: 16, marginBottom: 24 }}>
						<button
							onClick={() => handleConfigureCache(false)}
							style={{
								padding: "12px 24px",
								background: cacheMode === "stale" ? "#dc3545" : "#6c757d",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Stale Cache (Vulnerable)
						</button>
						<button
							onClick={() => handleConfigureCache(true)}
							style={{
								padding: "12px 24px",
								background: cacheMode === "smart" ? "#28a745" : "#6c757d",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Smart Cache (Protected)
						</button>
					</div>

					<h2>Step 2: Rotate Keys and Sign Token</h2>
					<div style={{ display: "flex", gap: 16 }}>
						<button
							onClick={() => handleRotateAndSign(false)}
							style={{
								padding: "12px 24px",
								background: "#ffc107",
								color: "black",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Rotate (keep old key)
						</button>
						<button
							onClick={() => handleRotateAndSign(true)}
							style={{
								padding: "12px 24px",
								background: "#dc3545",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Rotate + Revoke old key
						</button>
					</div>
				</div>
			)}

			{phase === "rotated" && (
				<div>
					<h2>Step 3: Verify Token</h2>
					{rotationInfo && (
						<div
							style={{
								background: "#f8f9fa",
								padding: 16,
								borderRadius: 8,
								marginBottom: 16,
							}}
						>
							<strong>Rotation:</strong> {rotationInfo}
						</div>
					)}
					<p>
						Cache mode:{" "}
						<strong>{cacheMode === "smart" ? "Smart" : "Stale"}</strong>
					</p>
					{token && (
						<div
							style={{
								background: "#e2e3e5",
								padding: 12,
								borderRadius: 8,
								marginBottom: 16,
							}}
						>
							<p>
								<strong>Signed Token:</strong>
							</p>
							<code style={{ wordBreak: "break-all", fontSize: 12 }}>
								{token}
							</code>
						</div>
					)}
					<button
						onClick={handleVerify}
						style={{
							padding: "12px 24px",
							background: "#007bff",
							color: "white",
							border: "none",
							borderRadius: 8,
							cursor: "pointer",
						}}
					>
						Verify Token
					</button>
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
							{result.defeated ? "BOSS DEFEATED!" : "Verification Failed!"}
						</strong>
						<p>{result.message}</p>
						{result.explanation && (
							<p style={{ color: "#666", whiteSpace: "pre-line" }}>
								{result.explanation}
							</p>
						)}
						{result.rfc_link && (
							<p>
								<a
									href={result.rfc_link}
									target="_blank"
									rel="noopener noreferrer"
								>
									RFC 7517 - JSON Web Key
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
