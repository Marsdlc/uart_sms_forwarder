import {useState} from 'react';
import {Filter, Loader2, Save, TestTube} from 'lucide-react';
import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {toast} from 'sonner';
import {Button} from '@/components/ui/button';
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from '@/components/ui/card';
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from '@/components/ui/select';
import {Switch} from '@/components/ui/switch';
import {Textarea} from '@/components/ui/textarea';
import {PageHeader} from '@/components/PageHeader';
import {
    getSMSFilterConfig,
    saveSMSFilterConfig,
    testSMSFilter,
    type SMSFilterConfig,
} from '@/api/property';

const DEFAULT_CONFIG: SMSFilterConfig = {enabled: false, mode: 'include', pattern: ''};

function errorMessage(error: unknown): string {
    if (!(error instanceof Error)) return '请求失败，请稍后重试';
    // API 客户端将后端 JSON 错误作为字符串返回，提取可读的校验信息。
    try {
        const body: unknown = JSON.parse(error.message);
        if (body && typeof body === 'object' && 'error' in body && typeof body.error === 'string') {
            return body.error;
        }
    } catch {
        // 网络或认证错误已经是普通文本。
    }
    return error.message;
}

export default function SMSFilterSettings() {
    const queryClient = useQueryClient();
    const [draft, setDraft] = useState<SMSFilterConfig | null>(null);
    const [sampleContent, setSampleContent] = useState('');
    const configQuery = useQuery({queryKey: ['smsFilterConfig'], queryFn: getSMSFilterConfig});
    const savedValues = configQuery.data ?? DEFAULT_CONFIG;
    const formValues = draft ?? savedValues;
    const isDirty = formValues.enabled !== savedValues.enabled ||
        formValues.mode !== savedValues.mode || formValues.pattern !== savedValues.pattern;

    const saveMutation = useMutation({
        mutationFn: saveSMSFilterConfig,
        onSuccess: (_data, config) => {
            queryClient.setQueryData(['smsFilterConfig'], config);
            setDraft(null);
            toast.success('短信过滤配置已保存');
            void queryClient.invalidateQueries({queryKey: ['smsFilterConfig']});
        },
        onError: (error: unknown) => toast.error(errorMessage(error)),
    });
    const testMutation = useMutation({
        mutationFn: testSMSFilter,
    });
    const testRequest = {config: formValues, content: sampleContent};
    // 草稿或测试正文变化后，隐藏此前的结果，包括仍在返回途中的旧请求。
    const resultIsCurrent = JSON.stringify(testMutation.variables) === JSON.stringify(testRequest);
    const testResult = resultIsCurrent && testMutation.isSuccess ? testMutation.data : undefined;
    const testError = resultIsCurrent && testMutation.isError ? errorMessage(testMutation.error) : undefined;

    const handleSave = () => {
        if (formValues.enabled && !formValues.pattern.trim()) {
            toast.warning('启用短信过滤时必须填写正则表达式');
            return;
        }
        saveMutation.mutate({...formValues});
    };

    if (configQuery.isPending) {
        return <div className="flex items-center justify-center py-20" role="status" aria-label="加载短信过滤配置">
            <Loader2 className="h-8 w-8 animate-spin text-blue-600"/>
        </div>;
    }

    if (configQuery.isError) {
        return <div className="space-y-4">
            <PageHeader title="短信过滤" description="配置短信转发前的正则过滤规则。"/>
            <Card><CardContent className="space-y-4">
                <p className="text-sm text-rose-600" role="alert">加载配置失败：{errorMessage(configQuery.error)}</p>
                <Button variant="outline" onClick={() => void configQuery.refetch()}>重新加载</Button>
            </CardContent></Card>
        </div>;
    }

    return (
        <div className="space-y-6 animate-in fade-in duration-300">
            <PageHeader title="短信过滤" description="使用正则表达式决定收到的短信是否转发到所有已启用的通知渠道。"/>
            <div className="grid grid-cols-1 gap-6 xl:grid-cols-2">
                <Card>
                    <CardHeader>
                        <CardTitle className="flex items-center gap-2 text-base">
                            <Filter className="h-5 w-5 text-blue-600"/>转发规则
                        </CardTitle>
                        <CardDescription>只匹配短信原始正文。被过滤的短信仍保留在短信中心。</CardDescription>
                    </CardHeader>
                    <CardContent className="space-y-6">
                        <div className="flex items-center justify-between gap-4 rounded-xl border border-slate-200 bg-slate-50 p-4">
                            <div>
                                <p className="font-medium text-slate-900">启用短信过滤</p>
                                <p className="mt-1 text-sm text-slate-500">关闭后，收到的短信均可转发。</p>
                            </div>
                            <Switch checked={formValues.enabled}
                                onCheckedChange={(enabled) => setDraft({...formValues, enabled})}
                                disabled={saveMutation.isPending} aria-label="启用短信过滤"/>
                        </div>
                        <div className="space-y-2">
                            <label htmlFor="sms-filter-mode" className="block text-sm font-medium text-slate-800">过滤模式</label>
                            <Select value={formValues.mode} disabled={saveMutation.isPending}
                                onValueChange={(mode) => {
                                    if (mode === 'include' || mode === 'exclude') setDraft({...formValues, mode});
                                }}>
                                <SelectTrigger id="sms-filter-mode" className="w-full"><SelectValue/></SelectTrigger>
                                <SelectContent>
                                    <SelectItem value="include">匹配后转发</SelectItem>
                                    <SelectItem value="exclude">匹配后拦截</SelectItem>
                                </SelectContent>
                            </Select>
                            <p className="text-xs text-slate-500">{formValues.mode === 'include'
                                ? '仅转发正文匹配正则的短信，其余短信不转发。'
                                : '正文匹配正则的短信不转发，其余短信正常转发。'}</p>
                        </div>
                        <div className="space-y-2">
                            <label htmlFor="sms-filter-pattern" className="block text-sm font-medium text-slate-800">正则表达式</label>
                            <Textarea id="sms-filter-pattern" value={formValues.pattern}
                                onChange={(event) => setDraft({...formValues, pattern: event.target.value})}
                                disabled={saveMutation.isPending} placeholder="验证码|校验码"
                                className="min-h-24 font-mono" spellCheck={false} aria-describedby="sms-filter-pattern-hint"/>
                            <p id="sms-filter-pattern-hint" className="text-xs leading-5 text-slate-500">
                                匹配正文中任意位置；用 ^ 和 $ 限定开头和结尾。直接填写表达式，无需 / 分隔符。
                                支持 (?i) 忽略大小写、(?s) 让点号匹配换行；不支持前后查找和反向引用。
                            </p>
                        </div>
                        <div className="flex items-center justify-between gap-3 border-t border-slate-100 pt-5">
                            <span className="text-xs text-slate-500">{isDirty ? '有未保存的修改' : '当前配置已保存'}</span>
                            <Button onClick={handleSave} disabled={!isDirty || saveMutation.isPending}>
                                {saveMutation.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin"/> : <Save className="mr-2 h-4 w-4"/>}
                                {saveMutation.isPending ? '保存中...' : '保存配置'}
                            </Button>
                        </div>
                    </CardContent>
                </Card>
                <div className="space-y-6">
                    <Card>
                        <CardHeader>
                            <CardTitle className="flex items-center gap-2 text-base">
                                <TestTube className="h-5 w-5 text-blue-600"/>测试规则
                            </CardTitle>
                            <CardDescription>使用左侧当前草稿测试，无需保存。测试不会发送通知或保存短信。</CardDescription>
                        </CardHeader>
                        <CardContent className="space-y-4">
                            <label htmlFor="sms-filter-sample" className="block text-sm font-medium text-slate-800">测试短信正文</label>
                            <Textarea id="sms-filter-sample" value={sampleContent}
                                onChange={(event) => setSampleContent(event.target.value)}
                                placeholder="请输入一条短信，查看是否会被转发" className="min-h-36"/>
                            <Button variant="outline" onClick={() => testMutation.mutate(testRequest)} disabled={testMutation.isPending}>
                                {testMutation.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin"/> : <TestTube className="mr-2 h-4 w-4"/>}
                                {testMutation.isPending ? '测试中...' : '测试当前规则'}
                            </Button>
                            <div aria-live="polite">
                                {testResult && <div className={`rounded-xl border p-4 text-sm ${testResult.forward
                                    ? 'border-emerald-200 bg-emerald-50 text-emerald-800'
                                    : 'border-amber-200 bg-amber-50 text-amber-800'}`}>
                                    <p className="font-semibold">{testResult.forward ? '允许转发' : '将被过滤，不转发'}</p>
                                    <p className="mt-1">{!formValues.enabled ? '过滤已关闭，短信直接放行。'
                                        : testResult.matched ? '正文匹配正则表达式。' : '正文未匹配正则表达式。'}</p>
                                </div>}
                                {testError && <p className="break-words text-sm text-rose-600" role="alert">{testError}</p>}
                            </div>
                        </CardContent>
                    </Card>
                    <Card>
                        <CardHeader><CardTitle className="text-base">使用说明</CardTitle></CardHeader>
                        <CardContent className="space-y-3 text-sm leading-6 text-slate-600">
                            <p>例如填写 <code className="rounded bg-slate-100 px-1.5 py-0.5 font-mono">验证码|校验码</code>，
                                选择“匹配后转发”可以只推送验证码短信。</p>
                            <p>来电、飞行模式、短信发送失败和通知渠道测试继续正常通知。</p>
                            <p>如果运行时配置读取失败或规则无效，该条短信会保留记录并跳过转发，原因记录在服务日志中。</p>
                        </CardContent>
                    </Card>
                </div>
            </div>
        </div>
    );
}
