'use client';

import { useState } from 'react';
import { sendMessage } from '../../api/handlers';

interface ComposeModalProps {
  isOpen: boolean;
  onClose: () => void;
  sessionToken?: string;
  senderAddress: string;
}

export const ComposeModal = ({
  isOpen,
  onClose,
  sessionToken,
  senderAddress,
}: ComposeModalProps) => {
  const [to, setTo] = useState('');
  const [cc, setCc] = useState('');
  const [subject, setSubject] = useState('');
  const [body, setBody] = useState('');
  const [files, setFiles] = useState<File[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);

  const handleSend = async () => {
    setError('');
    setIsLoading(true);

    try {
      // Validate form
      if (!to.trim()) {
        throw new Error('Recipient required');
      }

      if (!sessionToken) throw new Error('No session token');

      // Upload attachments (Phase 2e)
      const attachmentIds: { id: string; name: string; size: number; sha256: string }[] = [];

      for (const file of files) {
        try {
          const formData = new FormData();
          formData.append('file', file);

          const response = await fetch('http://localhost:6001/api/content/upload', {
            method: 'POST',
            headers: {
              'Authorization': `Bearer ${sessionToken}`,
            },
            body: formData,
          });

          if (!response.ok) {
            throw new Error(`Upload failed: ${response.status}`);
          }

          const uploadResult = await response.json() as { id: string; sha256: string };
          attachmentIds.push({
            id: uploadResult.id,
            name: file.name,
            size: file.size,
            sha256: uploadResult.sha256,
          });
        } catch (err) {
          throw new Error(`Failed to upload ${file.name}: ${err instanceof Error ? err.message : 'unknown error'}`);
        }
      }

      // Build recipients list
      const recipients = [
        ...to.split(',').map((s) => s.trim()).filter(Boolean),
        ...cc.split(',').map((s) => s.trim()).filter(Boolean),
      ];

      // Build envelope with attachments
      const envelope = {
        v: 'ucp/1.0',
        from: senderAddress,
        to: recipients,
        cc: cc ? cc.split(',').map((s) => s.trim()) : [],
        thread_id: generateULID(),
        signing_key: 'placeholder-key', // In real app: from identity manager
        body: {
          blocks: [
            {
              type: 'text',
              content: body,
            },
            ...attachmentIds.map((att) => ({
              type: 'attachment',
              content_id: att.id,
              filename: att.name,
              size: att.size,
              sha256: att.sha256,
            })),
          ],
        },
        metadata: {
          subject,
          timestamp: Date.now(),
          attachments: attachmentIds.length,
        },
      };

      // Send message
      const result = await sendMessage(envelope, sessionToken);
      if (result.error) {
        throw new Error(result.error);
      }

      // Success
      setSuccess(true);
      setTimeout(() => {
        onClose();
        resetForm();
      }, 1500);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send message');
    } finally {
      setIsLoading(false);
    }
  };

  const resetForm = () => {
    setTo('');
    setCc('');
    setSubject('');
    setBody('');
    setFiles([]);
    setError('');
    setSuccess(false);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-[#111113] border border-[#1E1E22] rounded-lg w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 bg-[#111113] px-6 py-4 border-b border-[#1E1E22] flex justify-between items-center">
          <h2 className="text-lg font-semibold text-[#FAFAFA]">Compose Message</h2>
          <button
            onClick={onClose}
            className="text-[#52525B] hover:text-[#FAFAFA] text-xl"
            disabled={isLoading}
          >
            ✕
          </button>
        </div>

        {/* Form */}
        <div className="p-6 space-y-4">
          {success && (
            <div className="bg-green-900/20 border border-green-600 text-green-400 p-3 rounded text-sm">
              ✓ Message sent successfully
            </div>
          )}

          {error && (
            <div className="bg-red-900/20 border border-red-600 text-red-400 p-3 rounded text-sm">
              {error}
            </div>
          )}

          {/* From (read-only) */}
          <div>
            <label className="block text-xs font-semibold text-[#FAFAFA] mb-2">
              From
            </label>
            <input
              type="text"
              value={senderAddress}
              disabled
              className="w-full px-3 py-2 bg-[#18181B] border border-[#1E1E22] rounded text-[12px] text-[#52525B] cursor-not-allowed"
            />
          </div>

          {/* To */}
          <div>
            <label className="block text-xs font-semibold text-[#FAFAFA] mb-2">
              To *
            </label>
            <input
              type="email"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              placeholder="recipient@example.com (comma-separated for multiple)"
              className="w-full px-3 py-2 bg-[#18181B] border border-[#1E1E22] rounded text-[12px] text-[#FAFAFA] placeholder-[#52525B] focus:outline-none focus:border-[#6366F1]"
              disabled={isLoading}
            />
          </div>

          {/* CC */}
          <div>
            <label className="block text-xs font-semibold text-[#FAFAFA] mb-2">
              Cc
            </label>
            <input
              type="text"
              value={cc}
              onChange={(e) => setCc(e.target.value)}
              placeholder="cc@example.com (optional)"
              className="w-full px-3 py-2 bg-[#18181B] border border-[#1E1E22] rounded text-[12px] text-[#FAFAFA] placeholder-[#52525B] focus:outline-none focus:border-[#6366F1]"
              disabled={isLoading}
            />
          </div>

          {/* Subject */}
          <div>
            <label className="block text-xs font-semibold text-[#FAFAFA] mb-2">
              Subject
            </label>
            <input
              type="text"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="Message subject"
              className="w-full px-3 py-2 bg-[#18181B] border border-[#1E1E22] rounded text-[12px] text-[#FAFAFA] placeholder-[#52525B] focus:outline-none focus:border-[#6366F1]"
              disabled={isLoading}
            />
          </div>

          {/* Body */}
          <div>
            <label className="block text-xs font-semibold text-[#FAFAFA] mb-2">
              Message
            </label>
            <textarea
              value={body}
              onChange={(e) => setBody(e.target.value)}
              placeholder="Your message here..."
              rows={8}
              className="w-full px-3 py-2 bg-[#18181B] border border-[#1E1E22] rounded text-[12px] text-[#FAFAFA] placeholder-[#52525B] focus:outline-none focus:border-[#6366F1] font-mono resize-none"
              disabled={isLoading}
            />
          </div>

          {/* Attachments (Phase 2e) */}
          <div>
            <label className="block text-xs font-semibold text-[#FAFAFA] mb-2">
              Attachments (Phase 2e - encrypted upload)
            </label>
            <input
              type="file"
              multiple
              onChange={(e) => setFiles(Array.from(e.target.files || []))}
              className="w-full text-[12px] text-[#52525B]"
              disabled={isLoading}
            />
            {files.length > 0 && (
              <div className="mt-2 space-y-1">
                {files.map((f, i) => (
                  <div key={i} className="text-[11px] text-[#52525B]">
                    {f.name} ({Math.round(f.size / 1024)} KB)
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="sticky bottom-0 bg-[#111113] px-6 py-4 border-t border-[#1E1E22] flex gap-3 justify-end">
          <button
            onClick={onClose}
            disabled={isLoading}
            className="px-4 py-2 text-[12px] font-semibold text-[#FAFAFA] bg-[#18181B] border border-[#1E1E22] rounded hover:bg-[#27272A] disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Cancel
          </button>
          <button
            onClick={handleSend}
            disabled={isLoading || !to.trim()}
            className="px-4 py-2 text-[12px] font-semibold text-white bg-[#6366F1] rounded hover:bg-[#4F46E5] disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {isLoading && <span className="animate-spin">⟳</span>}
            {success ? 'Sent!' : 'Send'}
          </button>
        </div>
      </div>
    </div>
  );
};

// Generate ULID-like ID (simplified)
function generateULID(): string {
  const timestamp = Date.now().toString(36).padStart(10, '0');
  const random = Math.random().toString(36).substring(2, 16).padStart(6, '0');
  return `${timestamp}${random}`.toUpperCase();
}
