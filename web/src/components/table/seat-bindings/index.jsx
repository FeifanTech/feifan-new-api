/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useState } from 'react';
import {
  Banner,
  Button,
  Form,
  Input,
  Modal,
  Popconfirm,
  Space,
  Table,
  Tag,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';
import CardPro from '../../common/ui/CardPro';

const SeatBindingsTable = () => {
  const { t } = useTranslation();
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [visible, setVisible] = useState(false);
  const [formApi, setFormApi] = useState(null);
  const [oauthLoading, setOauthLoading] = useState(false);
  const [oauthFlow, setOauthFlow] = useState(null);

  const loadData = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/seat_binding');
      const { success, data: payload, message } = res.data || {};
      if (!success) {
        showError(message || t('加载 Seat 绑定失败'));
        return;
      }
      const items = payload?.items || payload?.data?.items || [];
      setData(items);
    } catch (e) {
      showError(e?.message || t('加载 Seat 绑定失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const onCreate = async () => {
    const values = formApi?.getValues?.() || {};
    const userId = Number(values.user_id);
    if (!Number.isInteger(userId) || userId <= 0 || !values.seat_id || !values.github_token) {
      showError(t('user_id、seat_id、github_token 为必填'));
      return;
    }
    const payload = {
      ...values,
      user_id: userId,
      tenant_id: (values.tenant_id || '').trim(),
      seat_id: (values.seat_id || '').trim(),
      github_token: (values.github_token || '').trim(),
    };
    setLoading(true);
    try {
      const res = await API.post('/api/seat_binding', payload);
      const { success, message } = res.data || {};
      if (!success) {
        showError(message || t('保存 Seat 绑定失败'));
        return;
      }
      showSuccess(t('Seat 绑定已保存'));
      setVisible(false);
      formApi?.reset?.();
      await loadData();
    } catch (e) {
      showError(e?.message || t('保存 Seat 绑定失败'));
    } finally {
      setLoading(false);
    }
  };

  const onStartGitHubOAuth = async () => {
    const values = formApi?.getValues?.() || {};
    const userId = Number(values.user_id);
    if (!Number.isInteger(userId) || userId <= 0 || !values.seat_id) {
      showError(t('请先填写有效的 user_id 和 seat_id'));
      return;
    }
    setOauthLoading(true);
    try {
      const res = await API.post('/api/seat_binding/oauth/github/device/start', {
        tenant_id: (values.tenant_id || '').trim(),
        user_id: userId,
        seat_id: (values.seat_id || '').trim(),
        account_type: 'github_oauth',
      });
      const { success, message, data: flow } = res.data || {};
      if (!success) {
        showError(message || t('启动 GitHub OAuth 失败'));
        return;
      }
      setOauthFlow(flow);
      showSuccess(t('已生成授权码，请在 GitHub 页面完成授权'));
    } catch (e) {
      showError(e?.message || t('启动 GitHub OAuth 失败'));
    } finally {
      setOauthLoading(false);
    }
  };

  const onPollGitHubOAuth = async () => {
    if (!oauthFlow?.device_code) {
      showError(t('请先启动 GitHub OAuth'));
      return;
    }
    setOauthLoading(true);
    try {
      const res = await API.post('/api/seat_binding/oauth/github/device/poll', {
        device_code: oauthFlow.device_code,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('轮询 GitHub OAuth 失败'));
        return;
      }
      if (data?.status === 'pending' || data?.status === 'slow_down') {
        showSuccess(t('授权尚未完成，请在 GitHub 页面确认后重试'));
        return;
      }
      if (data?.status === 'bound') {
        showSuccess(t('GitHub OAuth 绑定成功'));
        setOauthFlow(null);
        setVisible(false);
        formApi?.reset?.();
        await loadData();
      }
    } catch (e) {
      showError(e?.message || t('轮询 GitHub OAuth 失败'));
    } finally {
      setOauthLoading(false);
    }
  };

  const onDelete = async (id) => {
    setLoading(true);
    try {
      const res = await API.delete(`/api/seat_binding/${id}`);
      const { success, message } = res.data || {};
      if (!success) {
        showError(message || t('删除 Seat 绑定失败'));
        return;
      }
      showSuccess(t('Seat 绑定已删除'));
      await loadData();
    } catch (e) {
      showError(e?.message || t('删除 Seat 绑定失败'));
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: 'Tenant', dataIndex: 'tenant_id' },
    { title: 'User ID', dataIndex: 'user_id', width: 120 },
    { title: 'Seat ID', dataIndex: 'seat_id' },
    {
      title: t('账号类型'),
      dataIndex: 'account_type',
      width: 120,
      render: (v) => <Tag>{v || 'individual'}</Tag>,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 120,
      render: (v) => (
        <Tag color={v === 'active' ? 'green' : 'grey'}>{v || 'active'}</Tag>
      ),
    },
    {
      title: t('操作'),
      dataIndex: 'operate',
      width: 120,
      render: (_, record) => (
        <Popconfirm
          title={t('确认删除该 Seat 绑定？')}
          onConfirm={() => onDelete(record.id)}
        >
          <Button type='danger' theme='borderless'>
            {t('删除')}
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <>
      <Modal
        title={t('新建 Seat 绑定')}
        visible={visible}
        onCancel={() => {
          setVisible(false);
          setOauthFlow(null);
        }}
        onOk={onCreate}
        okText={t('保存')}
        cancelText={t('取消')}
        confirmLoading={loading}
      >
        <Form getFormApi={setFormApi} labelPosition='left'>
          <Form.Input field='tenant_id' label='Tenant ID' placeholder='default' />
          <Form.Input field='user_id' label='User ID' />
          <Form.Input field='seat_id' label='Seat ID' />
          <Form.Input field='github_token' label='GitHub Token' mode='password' />
          <Form.Input field='account_type' label='Account Type' placeholder='individual/business/enterprise' />
        </Form>
        <Banner
          type='info'
          description={t('如果你使用 Copilot Free，建议使用下方 GitHub OAuth 设备授权（无需手填 PAT）。')}
          closeIcon={null}
          style={{ marginTop: 12 }}
        />
        <Space style={{ marginTop: 8 }}>
          <Button loading={oauthLoading} onClick={onStartGitHubOAuth}>
            {t('启动 GitHub OAuth 绑定')}
          </Button>
          <Button loading={oauthLoading} type='primary' theme='solid' onClick={onPollGitHubOAuth}>
            {t('我已授权，完成绑定')}
          </Button>
        </Space>
        {oauthFlow?.verification_uri && oauthFlow?.user_code ? (
          <Banner
            type='warning'
            closeIcon={null}
            style={{ marginTop: 12 }}
            description={
              <div>
                <div>{t('1) 打开链接：')}{oauthFlow.verification_uri}</div>
                <div>{t('2) 输入用户码：')}{oauthFlow.user_code}</div>
                <div>{t('3) 完成后点击“我已授权，完成绑定”')}</div>
              </div>
            }
          />
        ) : null}
      </Modal>

      <CardPro
        type='type1'
        actionsArea={
          <Space>
            <Button onClick={loadData}>{t('刷新')}</Button>
            <Button theme='solid' type='primary' onClick={() => setVisible(true)}>
              {t('新增 Seat 绑定')}
            </Button>
          </Space>
        }
        t={t}
      >
        <Table
          rowKey='id'
          loading={loading}
          dataSource={data}
          columns={columns}
          pagination={false}
        />
      </CardPro>
    </>
  );
};

export default SeatBindingsTable;

