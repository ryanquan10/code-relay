import React, { useMemo, useState } from 'react';
import { publicImportAPI } from '../api/client';

interface ProviderInfoPreview {
  sourceType: 'codex' | 'claude';
  upstreamURL: string;
  upstreamToken: string;
}

const PublicImport: React.FC = () => {
  const [platform] = useState<'xianyu'>('xianyu');
  const [text, setText] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [resultMsg, setResultMsg] = useState<string | null>(null);

  const previews = useMemo<ProviderInfoPreview[]>(() => parseText(text), [text]);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setResultMsg(null);
    try {
      const res = await publicImportAPI.importSources(platform, text);
      if (res.success) {
        setResultMsg(
          `导入成功：新建来源 ${res.created_sources} 条，新增映射 ${res.added_mappings} 条。`
        );
      } else {
        setResultMsg('未匹配到有效的 codex/claude 信息');
      }
    } catch (err: any) {
      setResultMsg(err?.response?.data?.error || err?.message || '提交失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div style={{ maxWidth: 900, margin: '24px auto', padding: 16 }}>
      <h2>公开导入（无需登录）</h2>
      <p style={{ color: '#666' }}>
        说明：仅支持从粘贴的文本中提取 codex/claude 的「来源类型、URL、Token」。
        平台固定为「xianyu」。将创建账号来源，并自动为同类型（codex/claude）的所有来源建立产品映射。
      </p>

      <form onSubmit={onSubmit}>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', fontWeight: 600, marginBottom: 6 }}>平台</label>
          <select value={platform} onChange={() => {}} disabled style={{ padding: 6 }}>
            <option value="xianyu">xianyu</option>
          </select>
        </div>

        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', fontWeight: 600, marginBottom: 6 }}>文本内容</label>
          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            placeholder="粘贴包含 codex/claude、URL、Token 的文本"
            rows={10}
            style={{ width: '100%', fontFamily: 'monospace', padding: 8 }}
          />
        </div>

        <button type="submit" disabled={submitting || !text.trim()} style={{ padding: '8px 16px' }}>
          {submitting ? '提交中…' : '提交导入'}
        </button>
      </form>

      <div style={{ marginTop: 16 }}>
        {resultMsg && <div style={{ padding: 10, background: '#f6ffed', border: '1px solid #b7eb8f' }}>{resultMsg}</div>}
      </div>

      <div style={{ marginTop: 24 }}>
        <h3>解析预览</h3>
        {previews.length === 0 ? (
          <div style={{ color: '#999' }}>暂无匹配</div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={th}>类型</th>
                <th style={th}>URL</th>
                <th style={th}>Token（仅展示前后段）</th>
              </tr>
            </thead>
            <tbody>
              {previews.map((p, i) => (
                <tr key={i}>
                  <td style={td}>{p.sourceType}</td>
                  <td style={td}><code>{p.upstreamURL}</code></td>
                  <td style={td}><code>{mask(p.upstreamToken)}</code></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
};

const th: React.CSSProperties = { textAlign: 'left', borderBottom: '1px solid #eee', padding: '6px 4px' };
const td: React.CSSProperties = { borderBottom: '1px solid #f5f5f5', padding: '6px 4px', verticalAlign: 'top' };

function mask(t: string) {
  if (!t) return '';
  if (t.length <= 12) return t;
  return t.slice(0, 6) + '…' + t.slice(-6);
}

function parseText(text: string): ProviderInfoPreview[] {
  const results: ProviderInfoPreview[] = [];
  const lines = text.trim().split(/\n+/).map((l) => l.trim()).filter(Boolean);

  // 模式1：三行
  for (let i = 0; i + 2 < lines.length; i++) {
    const l1 = lines[i];
    const l2 = lines[i + 1];
    const l3 = lines[i + 2];
    const src = normalizeSource(l1);
    if (!src) continue;
    const u = extractURL(l2);
    if (!u) continue;
    const t = extractToken(l3);
    if (!t) continue;
    results.push({ sourceType: src, upstreamURL: u, upstreamToken: t });
    i += 2;
  }

    // 模式1b：同一行包含 codex/claude 与 URL；下一行或下两行出现 Token
  for (let i = 0; i < lines.length; i++) {
    const l = lines[i];
    const src = normalizeSource(l);
    if (!src) continue;
    const u = extractURL(l);
    if (!u) continue;
    for (let j = 1; j <= 2 && i + j < lines.length; j++) {
      const t = extractToken(lines[i + j]);
      if (t) {
        results.push({ sourceType: src, upstreamURL: u, upstreamToken: t });
        i += j;
        break;
      }
    }
  }// 模式2：单行
  const reCombined = /(codex|claude)\s*([a-z][a-z0-9+.-]*:\/\/[^\s"']+?)(?:\s+|\s*[，,:;]\s*)([a-z0-9_\-]{20,}|(?:cr_)?[a-f0-9]{40,})/gi;
  let m: RegExpExecArray | null;
  while ((m = reCombined.exec(text)) !== null) {
    const src2 = normalizeSource(m[1]);
    if (!src2) continue;
    const u = m[2];
    const t = m[3];
    results.push({ sourceType: src2, upstreamURL: u, upstreamToken: t });
  }

  // 模式3：base_url + key
  const reBaseURL = /base_url\s*=\s*["']?([^"'\s]+)["']?/gi;
  const reKey = /"(?:OPENAI_API_KEY|API_KEY)"\s*:\s*["']?([^"'\s]+)["']?/gi;
  const urls: string[] = [];
  const keys: string[] = [];
  let mUrl: RegExpExecArray | null;
  while ((mUrl = reBaseURL.exec(text)) !== null) urls.push(mUrl[1]);
  let mKey: RegExpExecArray | null;
  while ((mKey = reKey.exec(text)) !== null) keys.push(mKey[1]);
  if (urls.length && keys.length) {
    for (const u of urls) {
      for (const k of keys) {
        if (/codex/i.test(u) || /codex/i.test(k)) {
          results.push({ sourceType: 'codex', upstreamURL: u, upstreamToken: k });
          break;
        }
      }
    }
  }

  // 去重(URL+Token)
  const seen = new Set<string>();
  return results.filter((r) => {
    const key = r.upstreamURL.toLowerCase() + '|' + r.upstreamToken;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  }) as ProviderInfoPreview[];
}

function normalizeSource(s: string): 'codex' | 'claude' | null {
  const v = s.trim().toLowerCase().replace(/^['"]|['"]$/g, '');
  if (v === 'codex' || v.includes('codex')) return 'codex';
  if (v === 'claude' || v.includes('claude')) return 'claude';
  return null;
}

function extractURL(s: string): string | null {
  const re = /(https?:\/\/[^\s"'><]+)/;
  const m = s.match(re);
  if (m) return m[1];
  const reLoose = /https?:\/\/[a-zA-Z0-9_.-]+(?:\/[a-zA-Z0-9_.\/?-]*)?/;
  const m2 = s.match(reLoose);
  return m2 ? m2[0] : null;
}

function extractToken(s: string): string | null {
  const re = /(?:cr_)?[A-Za-z0-9_\-]{40,}/;
  const m = s.match(re);
  return m ? m[0] : null;
}

export default PublicImport;




