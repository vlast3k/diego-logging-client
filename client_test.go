package diego_logging_client_test

import (
	"os"
	"path"
	"path/filepath"
	"time"

	client "code.cloudfoundry.org/diego-logging-client"
	"code.cloudfoundry.org/diego-logging-client/testhelpers"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	fixturesPath         = path.Join(os.Getenv("DIEGO_RELEASE_DIR"), "src/code.cloudfoundry.org/diego-logging-client/fixtures")
	metronCAFile         = path.Join(fixturesPath, "metron", "CA.crt")
	metronServerCertFile = path.Join(fixturesPath, "metron", "metron.crt")
	metronServerKeyFile  = path.Join(fixturesPath, "metron", "metron.key")
	metronClientCertFile = path.Join(fixturesPath, "metron", "client.crt")
	metronClientKeyFile  = path.Join(fixturesPath, "metron", "client.key")
)

var _ = Describe("DiegoLoggingClient", func() {
	var (
		c client.IngressClient
		fixturesPath,
		metronCAFile,
		metronServerCertFile,
		metronServerKeyFile,
		metronClientCertFile,
		metronClientKeyFile string
	)
	BeforeEach(func() {
		wd, err := os.Getwd()
		Expect(err).To(Succeed())
		fixturesPath = filepath.Join(wd, "fixtures")
		metronCAFile = filepath.Join(fixturesPath, "metron", "CA.crt")
		metronServerCertFile = filepath.Join(fixturesPath, "metron", "metron.crt")
		metronServerKeyFile = filepath.Join(fixturesPath, "metron", "metron.key")
		metronClientCertFile = filepath.Join(fixturesPath, "metron", "client.crt")
		metronClientKeyFile = filepath.Join(fixturesPath, "metron", "client.key")
	})

	Context("when the v2 api is used", func() {
		var (
			testIngressServer *testhelpers.TestIngressServer
			metricsPort       int
		)

		BeforeEach(func() {
			var err error

			testIngressServer, err = testhelpers.NewTestIngressServer(metronServerCertFile, metronServerKeyFile, metronCAFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(testIngressServer.Start()).To(Succeed())

			metricsPort, err = testIngressServer.Port()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("and the loggregator agent isn't up", func() {
			BeforeEach(func() {
				testIngressServer.Stop()
			})

			It("returns an error when constructing the loggregator client", func() {
				metricsPort := 8080

				_, err := client.NewIngressClient(client.Config{
					SourceID:           "some-source-id",
					InstanceID:         "some-instance-id",
					BatchFlushInterval: 10 * time.Millisecond,
					BatchMaxSize:       1,
					UseV2API:           true,
					APIPort:            metricsPort,
					CACertPath:         metronCAFile,
					KeyPath:            metronClientKeyFile,
					CertPath:           metronClientCertFile,
				})
				Expect(err).To(HaveOccurred())
			})
		})

		Context("and the loggregator agent is up", func() {
			var sender loggregator_v2.Ingress_BatchSenderServer

			BeforeEach(func() {
				sender = nil

				var err error
				c, err = client.NewIngressClient(client.Config{
					SourceID:           "some-source-id",
					InstanceID:         "some-instance-id",
					BatchFlushInterval: 10 * time.Millisecond,
					BatchMaxSize:       1,
					UseV2API:           true,
					APIPort:            metricsPort,
					CACertPath:         metronCAFile,
					KeyPath:            metronClientKeyFile,
					CertPath:           metronClientCertFile,
					JobOrigin:          "some-origin",
				})
				Expect(err).NotTo(HaveOccurred())
			})

			getEnvelopeBatch := func() *loggregator_v2.EnvelopeBatch {
				if sender == nil {
					Eventually(testIngressServer.Receivers()).Should(Receive(&sender))
				}
				batch, err := sender.Recv()
				Expect(err).NotTo(HaveOccurred())
				return batch
			}

			assertEnvelopeSourceAndInstanceIDAreCorrect := func(batch *loggregator_v2.EnvelopeBatch) {
				Expect(batch.Batch).To(HaveLen(1))
				Expect(batch.Batch[0].GetSourceId()).To(Equal("some-source-id"))
				Expect(batch.Batch[0].GetInstanceId()).To(Equal("some-instance-id"))
			}

			Describe("SendDuration", func() {
				It("sets app info", func() {
					Expect(c.SendDuration("time", 18*time.Second)).To(Succeed())

					assertEnvelopeSourceAndInstanceIDAreCorrect(getEnvelopeBatch())
				})
			})

			Describe("SendMebiBytes", func() {
				It("sets app info", func() {
					Expect(c.SendMebiBytes("disk-free", 23)).To(Succeed())

					assertEnvelopeSourceAndInstanceIDAreCorrect(getEnvelopeBatch())
				})
			})

			Describe("SendMetric", func() {
				It("sets app info", func() {
					Expect(c.SendMetric("errors", 3)).To(Succeed())

					assertEnvelopeSourceAndInstanceIDAreCorrect(getEnvelopeBatch())
				})
			})

			Describe("SendBytesPerSecond", func() {
				It("sets app info", func() {
					Expect(c.SendBytesPerSecond("speed", 3)).To(Succeed())

					assertEnvelopeSourceAndInstanceIDAreCorrect(getEnvelopeBatch())
				})
			})

			Describe("SendRequestsPerSecond", func() {
				It("sets app info", func() {
					Expect(c.SendRequestsPerSecond("homepage", 37)).To(Succeed())

					assertEnvelopeSourceAndInstanceIDAreCorrect(getEnvelopeBatch())
				})
			})

			Describe("IncrementCounter", func() {
				It("sets app info", func() {
					Expect(c.IncrementCounter("its")).To(Succeed())

					assertEnvelopeSourceAndInstanceIDAreCorrect(getEnvelopeBatch())
				})
			})

			Describe("IncrementCounterWithDelta", func() {
				It("sets app info", func() {
					Expect(c.IncrementCounterWithDelta("its", 5)).To(Succeed())

					assertEnvelopeSourceAndInstanceIDAreCorrect(getEnvelopeBatch())
				})
			})

			Describe("SendComponentMetric", func() {
				It("sets app info", func() {
					Expect(c.SendComponentMetric("memory", 37, "GB")).To(Succeed())

					assertEnvelopeSourceAndInstanceIDAreCorrect(getEnvelopeBatch())
				})
			})

			Describe("SendAppLog", func() {
				var batch *loggregator_v2.EnvelopeBatch

				JustBeforeEach(func() {
					tags := map[string]string{
						"source_id":   "some-source-id",
						"instance_id": "345",
						"some-key":    "some-value",
					}
					Expect(c.SendAppLog("some-message", "some-source-name", tags)).To(Succeed())
					batch = getEnvelopeBatch()
				})

				It("sets app info", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(batch.Batch[0].GetSourceId()).To(Equal("some-source-id"))
					Expect(batch.Batch[0].GetInstanceId()).To(Equal("345"))
				})

				It("sets tags", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(batch.Batch[0].GetTags()).To(Equal(map[string]string{
						"origin":      "some-origin",
						"source_id":   "some-source-id",
						"instance_id": "345",
						"source_type": "some-source-name",
						"some-key":    "some-value",
					}))
				})

				It("sets the correct type", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(batch.Batch[0].GetLog().Type).To(Equal(loggregator_v2.Log_OUT))
				})

				It("sets the message", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(string(batch.Batch[0].GetLog().GetPayload())).To(Equal("some-message"))
				})
			})

			Describe("SendAppErrorLog", func() {
				var batch *loggregator_v2.EnvelopeBatch

				JustBeforeEach(func() {
					tags := map[string]string{
						"source_id":   "some-source-id",
						"instance_id": "345",
						"some-key":    "some-value",
					}
					Expect(c.SendAppErrorLog("some-message", "some-source-name", tags)).To(Succeed())
					batch = getEnvelopeBatch()
				})

				It("sets app info", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(batch.Batch[0].GetSourceId()).To(Equal("some-source-id"))
					Expect(batch.Batch[0].GetInstanceId()).To(Equal("345"))
				})

				It("sets tags", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(batch.Batch[0].GetTags()).To(Equal(map[string]string{
						"origin":      "some-origin",
						"source_id":   "some-source-id",
						"instance_id": "345",
						"source_type": "some-source-name",
						"some-key":    "some-value",
					}))
				})

				It("sets the correct type", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(batch.Batch[0].GetLog().Type).To(Equal(loggregator_v2.Log_ERR))
				})

				It("sets the message", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(string(batch.Batch[0].GetLog().GetPayload())).To(Equal("some-message"))
				})
			})

			Describe("SendAppMetrics", func() {
				var batch *loggregator_v2.EnvelopeBatch

				JustBeforeEach(func() {
					metrics := client.ContainerMetric{
						MemoryBytes:      50,
						MemoryBytesQuota: 100,

						DiskBytes:      100,
						DiskBytesQuota: 200,

						CpuPercentage:          50.0,
						AbsoluteCPUUsage:       1,
						AbsoluteCPUEntitlement: 2,
						ContainerAge:           3,
						Tags: map[string]string{
							"source_id":   "some-source-id",
							"instance_id": "345",
							"some-key":    "some-value",
						},
					}

					Expect(c.SendAppMetrics(metrics)).To(Succeed())
					batch = getEnvelopeBatch()
				})

				It("sets app info", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(batch.Batch[0].GetSourceId()).To(Equal("some-source-id"))
					Expect(batch.Batch[0].GetInstanceId()).To(Equal("345"))
				})

				It("sends memory usage and quota", func() {
					metrics := batch.Batch[0].GetGauge().GetMetrics()
					Expect(metrics["memory"].GetValue()).To(Equal(float64(50)))
					Expect(metrics["memory"].GetUnit()).To(Equal("bytes"))

					Expect(metrics["memory_quota"].GetValue()).To(Equal(float64(100)))
					Expect(metrics["memory_quota"].GetUnit()).To(Equal("bytes"))
				})

				It("sends disk usage and quota", func() {
					metrics := batch.Batch[0].GetGauge().GetMetrics()
					Expect(metrics["disk"].GetValue()).To(Equal(float64(100)))
					Expect(metrics["disk"].GetUnit()).To(Equal("bytes"))

					Expect(metrics["disk_quota"].GetValue()).To(Equal(float64(200)))
					Expect(metrics["disk_quota"].GetUnit()).To(Equal("bytes"))
				})

				It("sends cpu usage in a separate batch", func() {
					batch = getEnvelopeBatch()

					metrics := batch.Batch[0].GetGauge().GetMetrics()

					Expect(metrics["absolute_usage"].GetValue()).To(Equal(float64(1)))
					Expect(metrics["absolute_usage"].GetUnit()).To(Equal("nanoseconds"))

					Expect(metrics["absolute_entitlement"].GetValue()).To(Equal(float64(2)))
					Expect(metrics["absolute_entitlement"].GetUnit()).To(Equal("nanoseconds"))

					Expect(metrics["container_age"].GetValue()).To(Equal(float64(3)))
					Expect(metrics["container_age"].GetUnit()).To(Equal("nanoseconds"))
				})

				It("sends tags", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(batch.Batch[0].GetTags()).To(Equal(map[string]string{
						"origin":      "some-origin",
						"source_id":   "some-source-id",
						"instance_id": "345",
						"some-key":    "some-value",
					}))
				})
			})

			Describe("SendAppLogRate", func() {
				var batch *loggregator_v2.EnvelopeBatch

				JustBeforeEach(func() {
					tags := map[string]string{
						"source_id":   "some-source-id",
						"instance_id": "345",
						"some-key":    "some-value",
					}
					Expect(c.SendAppLogRate(50, 100, tags)).To(Succeed())
					batch = getEnvelopeBatch()
				})

				It("sets app info", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(batch.Batch[0].GetSourceId()).To(Equal("some-source-id"))
					Expect(batch.Batch[0].GetInstanceId()).To(Equal("345"))
				})

				It("sends log rate and log rate limit", func() {
					metrics := batch.Batch[0].GetGauge().GetMetrics()
					Expect(metrics["log_rate"].GetValue()).To(Equal(float64(50)))
					Expect(metrics["log_rate"].GetUnit()).To(Equal("B/s"))

					Expect(metrics["log_rate_limit"].GetValue()).To(Equal(float64(100)))
					Expect(metrics["log_rate_limit"].GetUnit()).To(Equal("B/s"))
				})

				It("sends tags", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(batch.Batch[0].GetTags()).To(Equal(map[string]string{
						"origin":      "some-origin",
						"source_id":   "some-source-id",
						"instance_id": "345",
						"some-key":    "some-value",
					}))
				})
			})

			Describe("SendSpikeMetrics", func() {
				var batch *loggregator_v2.EnvelopeBatch

				JustBeforeEach(func() {
					metrics := client.SpikeMetric{
						Start: time.Unix(123, 0),
						End:   time.Unix(234, 0),
						Tags: map[string]string{
							"source_id":   "some-source-id",
							"instance_id": "345",
							"some-key":    "some-value",
						},
					}

					Expect(c.SendSpikeMetrics(metrics)).To(Succeed())
					batch = getEnvelopeBatch()
				})

				It("sets app info", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(batch.Batch[0].GetSourceId()).To(Equal("some-source-id"))
					Expect(batch.Batch[0].GetInstanceId()).To(Equal("345"))
				})

				It("sends start and end times", func() {
					metrics := batch.Batch[0].GetGauge().GetMetrics()
					Expect(metrics["spike_start"].GetValue()).To(Equal(float64(123)))
					Expect(metrics["spike_start"].GetUnit()).To(Equal("seconds"))

					Expect(metrics["spike_end"].GetValue()).To(Equal(float64(234)))
					Expect(metrics["spike_end"].GetUnit()).To(Equal("seconds"))
				})

				It("sends tags", func() {
					Expect(batch.Batch).To(HaveLen(1))
					Expect(batch.Batch[0].GetTags()).To(Equal(map[string]string{
						"origin":      "some-origin",
						"source_id":   "some-source-id",
						"instance_id": "345",
						"some-key":    "some-value",
					}))
				})
			})
		})
	})
})
