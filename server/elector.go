package server

import (
	"context"
	"time"

	pb "github.com/moooofly/dms-prosecutor/proto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func (p *Prosecutor) electorLoop() {
	logrus.Debug("====> enter electorLoop")
	defer logrus.Debug("====> leave electorLoop")

	go p.backgroundConnectElector()
	p.electorReconnectTrigger()

	ticker := time.NewTicker(time.Duration(p.checkPeriod) * time.Second)

	for {
		select {
		case <-p.stopCh:
			return

		case <-p.connectedElectorCh:
		}

		var curRole, prevRole pb.EnumRole

		for range ticker.C {

			logrus.Infof("[prosecutor]    --------")

			// step 1: 获取 elector role
			logrus.Infof("[prosecutor] --> send [Obtain] to elector")

			obRsp, err := p.rsClient.Obtain(context.Background(), &pb.ObtainReq{})
			if err != nil {
				logrus.Infof("[prosecutor] Obtain role failed: %v", err)
				p.electorReconnectTrigger()
				break
			}

			logrus.Infof("[prosecutor] <-- recv [Obtain] Resp, role => [%s]", obRsp.GetRole())

			// NOTE: 由于默认值的关系，preRole 初始值总是 Candidate
			prevRole, curRole = curRole, obRsp.GetRole()
			logrus.Infof("[prosecutor]     role status, last [%s] --> current [%s]", prevRole, curRole)

			// step 2: 发现自己所访问的 elector 从非 leader 变成 leader 则进行状态重置
			if curRole == pb.EnumRole_Leader && curRole != prevRole {
				// NOTE: fd, 20180717
				// every time the elector become the leader, reset the checklist's
				// to give it a chance to not to abdicate too early, cause it may need a long time
				// to start up services in the new leader side
				logrus.Infof("[prosecutor] reset serivce counter as of role update to [Leader]")

				for _, v := range p.allAppStatus {
					v.reset()
				}
			}

			// step 3: 基于 role 的类型进行状态检查
			if ok := p.checkAllServiceStatus(curRole); !ok {
				switch p.mode {
				case "single-point":
					logrus.Infof("[prosecutor] nothing to do in [single-point] mode")

				case "master-slave":
					logrus.Infof("[prosecutor] find service down, we should make elector abdicate in [master-slave] mode")

					// NOTE: 只有当 elector 是 leader 时，触发 abdicate 才有意义
					if curRole == pb.EnumRole_Leader {

						logrus.Infof("[prosecutor] --> send [Abdicate] to elector")

						abRsp, err := p.rsClient.Abdicate(context.Background(), &pb.AbdicateReq{})
						if err != nil {
							logrus.Infof("[prosecutor] Abdicate failed, reason: %v", err)
							p.electorReconnectTrigger()
							break
						}

						logrus.Infof("[prosecutor] <-- recv [Abdicate] Resp, [%s]", abRsp.String())
					} else {
						logrus.Infof("[prosecutor] elector role is [Follower], no need to abdicate")
					}

					// TODO: 可能需要根据 code 和 msg 做更多的逻辑判定

				case "cluster":
					logrus.Errorf("[prosecutor] cluster handler unfinished")

				default:
					logrus.Errorf("[prosecutor] wrong mode: %s", p.mode)
				}
			}

			logrus.Infof("[prosecutor]    --------")
		}
	}
}

func (p *Prosecutor) backgroundConnectElector() {
	for {
		select {
		case <-p.stopCh:
			return

		case <-p.disconnectedElectorCh:
		}

		for {

			logrus.Infof("[prosecutor] --> try to connect elector[%s]", p.rsTcpHost)

			conn, err := grpc.Dial(
				p.rsTcpHost,
				grpc.WithInsecure(),
				grpc.WithBlock(),
				grpc.WithTimeout(time.Second),
			)
			// NOTE: block + timeout
			if err != nil {
				logrus.Warnf("[prosecutor] connect elector failed, reason: %v", err)
			} else {
				logrus.Infof("[prosecutor] connect elector success")

				// NOTE: 顺序不能变
				p.rsClientConn = conn
				p.rsClient = pb.NewRoleServiceClient(conn)

				// FIXME: another way?
				p.connectedElectorCh <- struct{}{}

				break
			}

			time.Sleep(time.Second * time.Duration(p.reconnectPeriod))
		}
	}
}

func (p *Prosecutor) electorReconnectTrigger() {
	select {
	case p.disconnectedElectorCh <- struct{}{}:
		logrus.Debugf("[prosecutor] trigger connection to [elector]")
	default:
		logrus.Debugf("[prosecutor] connection process is ongoing")
	}
}

func (p *Prosecutor) disconnectElector() {
	if p.rsClientConn != nil {
		p.rsClientConn.Close()
	}
}
