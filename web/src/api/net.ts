import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import http from '@/lib/nos-client';
import type {
  NetworkStatus,
  FirewallState,
  FirewallPlan,
  WireGuardConfig,
  WireGuardPeerConfig,
  HTTPSConfig,
  TOTPEnrollment,
  TOTPStatus,
  RemoteAccessWizardState,
  EnableWireGuardRequest,
  AddWireGuardPeerRequest,
  ConfigureHTTPSRequest,
  PlanFirewallRequest,
  ApplyFirewallRequest,
  VerifyTOTPRequest,
  EnrollTOTPRequest,
} from './net.types';

// Network status
export function useNetworkStatus() {
  return useQuery<NetworkStatus>({
    queryKey: ['network', 'status'],
    queryFn: () => http.get('/v1/net/status'),
    refetchInterval: 10000, // Poll every 10 seconds
  });
}

// Firewall management
export function useFirewallState() {
  return useQuery<FirewallState>({
    queryKey: ['firewall', 'state'],
    queryFn: () => http.get('/v1/net/firewall/state'),
    refetchInterval: 5000, // Poll more frequently when confirming
  });
}

export function usePlanFirewall() {
  const queryClient = useQueryClient();
  
  return useMutation<FirewallPlan, Error, PlanFirewallRequest>({
    mutationFn: (data) => http.post('/v1/net/firewall/plan', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['firewall'] });
    },
  });
}

export function useApplyFirewall() {
  const queryClient = useQueryClient();
  
  return useMutation<void, Error, ApplyFirewallRequest>({
    mutationFn: (data) => http.post('/v1/net/firewall/apply', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['firewall'] });
      queryClient.invalidateQueries({ queryKey: ['network'] });
    },
  });
}

export function useConfirmFirewall() {
  const queryClient = useQueryClient();
  
  return useMutation<void, Error>({
    mutationFn: () => http.post('/v1/net/firewall/confirm'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['firewall'] });
      queryClient.invalidateQueries({ queryKey: ['network'] });
    },
  });
}

export function useRollbackFirewall() {
  const queryClient = useQueryClient();
  
  return useMutation<void, Error>({
    mutationFn: () => http.post('/v1/net/firewall/rollback'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['firewall'] });
      queryClient.invalidateQueries({ queryKey: ['network'] });
    },
  });
}

// WireGuard management
export function useWireGuardState() {
  return useQuery<WireGuardConfig>({
    queryKey: ['wireguard', 'state'],
    queryFn: () => http.get('/v1/net/wg/state'),
    refetchInterval: 30000, // Poll every 30 seconds
  });
}

export function useEnableWireGuard() {
  const queryClient = useQueryClient();
  
  return useMutation<void, Error, EnableWireGuardRequest>({
    mutationFn: (data) => http.post('/v1/net/wg/enable', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wireguard'] });
      queryClient.invalidateQueries({ queryKey: ['network'] });
    },
  });
}

export function useDisableWireGuard() {
  const queryClient = useQueryClient();
  
  return useMutation<void, Error>({
    mutationFn: () => http.post('/v1/net/wg/disable'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wireguard'] });
      queryClient.invalidateQueries({ queryKey: ['network'] });
    },
  });
}

export function useAddWireGuardPeer() {
  const queryClient = useQueryClient();
  
  return useMutation<WireGuardPeerConfig, Error, AddWireGuardPeerRequest>({
    mutationFn: (data) => http.post('/v1/net/wg/peers/add', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wireguard'] });
    },
  });
}

export function useRemoveWireGuardPeer() {
  const queryClient = useQueryClient();
  
  return useMutation<void, Error, string>({
    mutationFn: (peerId) => http.post(`/v1/net/wg/peers/remove?id=${peerId}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wireguard'] });
    },
  });
}

// HTTPS/ACME management
export function useHTTPSConfig() {
  return useQuery<HTTPSConfig>({
    queryKey: ['https', 'config'],
    queryFn: () => http.get('/v1/net/https/config'),
    refetchInterval: 60000, // Poll every minute
  });
}

export function useConfigureHTTPS() {
  const queryClient = useQueryClient();
  
  return useMutation<void, Error, ConfigureHTTPSRequest>({
    mutationFn: (data) => http.post('/v1/net/https/configure', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['https'] });
      queryClient.invalidateQueries({ queryKey: ['network'] });
    },
  });
}

export function useTestHTTPS() {
  return useMutation<{ status: string; message: string }, Error>({
    mutationFn: () => http.post('/v1/net/https/test'),
  });
}

// 2FA/TOTP management
export function useTOTPStatus() {
  return useQuery<TOTPStatus>({
    queryKey: ['auth', '2fa', 'status'],
    queryFn: () => http.get('/v1/auth/2fa/status'),
  });
}

export function useEnrollTOTP() {
  const queryClient = useQueryClient();
  
  return useMutation<TOTPEnrollment, Error, EnrollTOTPRequest>({
    mutationFn: (data) => http.post('/v1/auth/2fa/enroll', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth', '2fa'] });
    },
  });
}

export function useVerifyTOTP() {
  const queryClient = useQueryClient();
  
  return useMutation<void, Error, VerifyTOTPRequest>({
    mutationFn: (data) => http.post('/v1/auth/2fa/verify', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth', '2fa'] });
    },
  });
}

export function useDisableTOTP() {
  const queryClient = useQueryClient();
  
  return useMutation<void, Error>({
    mutationFn: () => http.post('/v1/auth/2fa/disable'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth', '2fa'] });
    },
  });
}

export function useRegenerateBackupCodes() {
  const queryClient = useQueryClient();
  
  return useMutation<{ backup_codes: string[] }, Error>({
    mutationFn: () => http.post('/v1/auth/2fa/backup-codes'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth', '2fa'] });
    },
  });
}

// Remote Access Wizard
export function useStartWizard() {
  const queryClient = useQueryClient();
  
  return useMutation<RemoteAccessWizardState, Error>({
    mutationFn: () => http.post('/v1/net/wizard/start'),
    onSuccess: (data) => {
      queryClient.setQueryData(['wizard', 'state'], data);
    },
  });
}

export function useWizardState() {
  return useQuery<RemoteAccessWizardState>({
    queryKey: ['wizard', 'state'],
    queryFn: () => http.get('/v1/net/wizard/state'),
    enabled: false, // Only fetch when needed
  });
}

export function useWizardNext() {
  const queryClient = useQueryClient();
  
  return useMutation<RemoteAccessWizardState, Error, Record<string, any>>({
    mutationFn: (data) => http.post('/v1/net/wizard/next', data),
    onSuccess: (data) => {
      queryClient.setQueryData(['wizard', 'state'], data);
    },
  });
}

export function useCompleteWizard() {
  const queryClient = useQueryClient();
  
  return useMutation<void, Error>({
    mutationFn: () => http.post('/v1/net/wizard/complete'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wizard'] });
      queryClient.invalidateQueries({ queryKey: ['network'] });
      queryClient.invalidateQueries({ queryKey: ['firewall'] });
      queryClient.invalidateQueries({ queryKey: ['wireguard'] });
      queryClient.invalidateQueries({ queryKey: ['https'] });
    },
  });
}
