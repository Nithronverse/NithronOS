import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import {
  Shield, Smartphone, Key, Copy, Download, RefreshCw,
  CheckCircle, XCircle, AlertTriangle, Loader2, QrCode
} from 'lucide-react';
import {
  useTOTPStatus,
  useEnrollTOTP,
  useVerifyTOTP,
  useDisableTOTP,
  useRegenerateBackupCodes,
} from '@/api/net';
import type { TOTPEnrollment } from '@/api/net.types';

export function TwoFactorSettings() {
  const [showEnrollDialog, setShowEnrollDialog] = useState(false);
  const [showVerifyDialog, setShowVerifyDialog] = useState(false);
  const [showBackupCodesDialog, setShowBackupCodesDialog] = useState(false);
  const [password, setPassword] = useState('');
  const [verificationCode, setVerificationCode] = useState('');
  const [enrollmentData, setEnrollmentData] = useState<TOTPEnrollment | null>(null);
  const [backupCodes, setBackupCodes] = useState<string[]>([]);

  const totpStatus = useTOTPStatus();
  const enrollTOTP = useEnrollTOTP();
  const verifyTOTP = useVerifyTOTP();
  const disableTOTP = useDisableTOTP();
  const regenerateBackupCodes = useRegenerateBackupCodes();

  const handleStartEnrollment = async () => {
    if (!password) return;

    try {
      const data = await enrollTOTP.mutateAsync({ password });
      setEnrollmentData(data);
      setShowEnrollDialog(false);
      setShowVerifyDialog(true);
      setPassword('');
    } catch (error) {
      console.error('Failed to enroll:', error);
    }
  };

  const handleVerifyEnrollment = async () => {
    if (!verificationCode || verificationCode.length !== 6) return;

    try {
      await verifyTOTP.mutateAsync({ code: verificationCode });
      setBackupCodes(enrollmentData?.backup_codes || []);
      setShowVerifyDialog(false);
      setShowBackupCodesDialog(true);
      setVerificationCode('');
      setEnrollmentData(null);
    } catch (error) {
      console.error('Failed to verify:', error);
    }
  };

  const handleDisable2FA = async () => {
    if (!confirm('Are you sure you want to disable two-factor authentication? This will make your account less secure.')) {
      return;
    }

    try {
      await disableTOTP.mutateAsync();
    } catch (error) {
      console.error('Failed to disable 2FA:', error);
    }
  };

  const handleRegenerateBackupCodes = async () => {
    if (!confirm('Are you sure you want to regenerate backup codes? Your old codes will no longer work.')) {
      return;
    }

    try {
      const result = await regenerateBackupCodes.mutateAsync();
      setBackupCodes(result.backup_codes);
      setShowBackupCodesDialog(true);
    } catch (error) {
      console.error('Failed to regenerate backup codes:', error);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const downloadBackupCodes = () => {
    const content = backupCodes.join('\n');
    const blob = new Blob([content], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'nithronos-backup-codes.txt';
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="container mx-auto p-6 max-w-4xl">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">Two-Factor Authentication</h1>
        <p className="text-muted-foreground">
          Add an extra layer of security to your account
        </p>
      </div>

      <div className="space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>2FA Status</CardTitle>
            <CardDescription>
              Current two-factor authentication configuration
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className={`p-2 rounded-full ${totpStatus.data?.enrolled ? 'bg-green-100 dark:bg-green-900' : 'bg-muted'}`}>
                  <Shield className={`h-5 w-5 ${totpStatus.data?.enrolled ? 'text-green-600 dark:text-green-400' : 'text-muted-foreground'}`} />
                </div>
                <div>
                  <div className="font-medium">Status</div>
                  <div className="text-sm text-muted-foreground">
                    {totpStatus.data?.enrolled ? 'Two-factor authentication is enabled' : 'Two-factor authentication is not enabled'}
                  </div>
                </div>
              </div>
              <div>
                {totpStatus.data?.enrolled ? (
                  <Badge variant="default">
                    <CheckCircle className="h-3 w-3 mr-1" />
                    Enabled
                  </Badge>
                ) : (
                  <Badge variant="secondary">
                    <XCircle className="h-3 w-3 mr-1" />
                    Disabled
                  </Badge>
                )}
              </div>
            </div>

            {totpStatus.data?.required && !totpStatus.data?.enrolled && (
              <Alert>
                <AlertTriangle className="h-4 w-4" />
                <AlertDescription>
                  Two-factor authentication is required for non-LAN access. Please enable it to secure your account.
                </AlertDescription>
              </Alert>
            )}

            <div className="flex gap-2">
              {!totpStatus.data?.enrolled ? (
                <Button onClick={() => setShowEnrollDialog(true)}>
                  <Smartphone className="h-4 w-4 mr-2" />
                  Enable 2FA
                </Button>
              ) : (
                <>
                  <Button variant="outline" onClick={handleRegenerateBackupCodes}>
                    <RefreshCw className="h-4 w-4 mr-2" />
                    Regenerate Backup Codes
                  </Button>
                  <Button variant="destructive" onClick={handleDisable2FA}>
                    Disable 2FA
                  </Button>
                </>
              )}
            </div>
          </CardContent>
        </Card>

        {totpStatus.data?.enrolled && (
          <Card>
            <CardHeader>
              <CardTitle>Session Status</CardTitle>
              <CardDescription>
                Current session verification status
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-2">
                {totpStatus.data?.verified ? (
                  <>
                    <CheckCircle className="h-5 w-5 text-green-600" />
                    <span>Your current session is 2FA verified</span>
                  </>
                ) : (
                  <>
                    <AlertTriangle className="h-5 w-5 text-yellow-600" />
                    <span>Your current session is not 2FA verified</span>
                  </>
                )}
              </div>
            </CardContent>
          </Card>
        )}

        <Card>
          <CardHeader>
            <CardTitle>How It Works</CardTitle>
            <CardDescription>
              Protect your account with time-based one-time passwords
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ol className="space-y-3">
              <li className="flex gap-3">
                <div className="flex-shrink-0 w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center text-sm font-medium">
                  1
                </div>
                <div>
                  <div className="font-medium">Install an authenticator app</div>
                  <div className="text-sm text-muted-foreground">
                    Use Google Authenticator, Authy, or any TOTP-compatible app
                  </div>
                </div>
              </li>
              <li className="flex gap-3">
                <div className="flex-shrink-0 w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center text-sm font-medium">
                  2
                </div>
                <div>
                  <div className="font-medium">Scan the QR code</div>
                  <div className="text-sm text-muted-foreground">
                    Link your authenticator app to your NithronOS account
                  </div>
                </div>
              </li>
              <li className="flex gap-3">
                <div className="flex-shrink-0 w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center text-sm font-medium">
                  3
                </div>
                <div>
                  <div className="font-medium">Save backup codes</div>
                  <div className="text-sm text-muted-foreground">
                    Store them securely in case you lose access to your device
                  </div>
                </div>
              </li>
            </ol>
          </CardContent>
        </Card>
      </div>

      {/* Enrollment Dialog - Step 1: Password */}
      <Dialog open={showEnrollDialog} onOpenChange={setShowEnrollDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Enable Two-Factor Authentication</DialogTitle>
            <DialogDescription>
              First, verify your password to continue
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label htmlFor="password">Current Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter your password"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEnrollDialog(false)}>
              Cancel
            </Button>
            <Button 
              onClick={handleStartEnrollment} 
              disabled={!password || enrollTOTP.isLoading}
            >
              {enrollTOTP.isLoading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              Continue
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Verification Dialog - Step 2: QR Code */}
      <Dialog open={showVerifyDialog} onOpenChange={setShowVerifyDialog}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Scan QR Code</DialogTitle>
            <DialogDescription>
              Scan this QR code with your authenticator app
            </DialogDescription>
          </DialogHeader>
          {enrollmentData && (
            <div className="space-y-4">
              <div className="flex justify-center p-4 bg-white rounded-lg">
                <img src={enrollmentData.qr_code} alt="TOTP QR Code" className="w-48 h-48" />
              </div>
              
              <div>
                <Label>Manual Entry</Label>
                <div className="mt-1 p-2 bg-muted rounded text-xs font-mono break-all">
                  {enrollmentData.secret}
                </div>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => copyToClipboard(enrollmentData.secret)}
                  className="mt-1"
                >
                  <Copy className="h-3 w-3 mr-1" />
                  Copy Secret
                </Button>
              </div>

              <div>
                <Label htmlFor="verification-code">Verification Code</Label>
                <Input
                  id="verification-code"
                  value={verificationCode}
                  onChange={(e) => setVerificationCode(e.target.value)}
                  placeholder="000000"
                  maxLength={6}
                  className="text-center text-2xl tracking-wider"
                />
                <p className="text-sm text-muted-foreground mt-1">
                  Enter the 6-digit code from your authenticator app
                </p>
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowVerifyDialog(false)}>
              Cancel
            </Button>
            <Button 
              onClick={handleVerifyEnrollment}
              disabled={verificationCode.length !== 6 || verifyTOTP.isLoading}
            >
              {verifyTOTP.isLoading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              Verify & Enable
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Backup Codes Dialog - Step 3: Save Codes */}
      <Dialog open={showBackupCodesDialog} onOpenChange={setShowBackupCodesDialog}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Save Your Backup Codes</DialogTitle>
            <DialogDescription>
              Store these codes in a safe place. Each code can only be used once.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <Alert>
              <Key className="h-4 w-4" />
              <AlertDescription>
                These codes are the only way to access your account if you lose your authenticator device.
                Treat them like passwords!
              </AlertDescription>
            </Alert>
            
            <div className="grid grid-cols-2 gap-2">
              {backupCodes.map((code, index) => (
                <div
                  key={index}
                  className="p-2 bg-muted rounded text-center font-mono text-sm"
                >
                  {code}
                </div>
              ))}
            </div>

            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={() => copyToClipboard(backupCodes.join('\n'))}
                className="flex-1"
              >
                <Copy className="h-4 w-4 mr-2" />
                Copy All
              </Button>
              <Button
                variant="outline"
                onClick={downloadBackupCodes}
                className="flex-1"
              >
                <Download className="h-4 w-4 mr-2" />
                Download
              </Button>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => setShowBackupCodesDialog(false)}>
              I've Saved My Codes
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
