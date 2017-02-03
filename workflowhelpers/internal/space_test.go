package internal_test

import (
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/config"
	"github.com/cloudfoundry-incubator/cf-test-helpers/internal/fakes"
	. "github.com/cloudfoundry-incubator/cf-test-helpers/workflowhelpers/internal"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("TestSpace", func() {
	var cfg config.Config
	var namePrefix string
	var quotaLimit string
	BeforeEach(func() {
		namePrefix = "UNIT-TEST"
		quotaLimit = "10G"
		cfg = config.Config{
			NamePrefix:   namePrefix,
			TimeoutScale: 1.0,
		}
	})

	Describe("NewRegularTestSpace", func() {
		It("generates a quotaDefinitionName", func() {
			testSpace := NewRegularTestSpace(&cfg, quotaLimit)
			Expect(testSpace.QuotaDefinitionName).To(MatchRegexp("%s-[0-9]-QUOTA-.*", namePrefix))
		})

		It("generates an organizationName", func() {
			testSpace := NewRegularTestSpace(&cfg, quotaLimit)
			Expect(testSpace.OrganizationName()).To(MatchRegexp("%s-[0-9]-ORG-.*", namePrefix))
		})

		It("generates a spaceName", func() {
			testSpace := NewRegularTestSpace(&cfg, quotaLimit)
			Expect(testSpace.SpaceName()).To(MatchRegexp("%s-[0-9]-SPACE-.*", namePrefix))
		})

		It("sets a timeout for cf commands", func() {
			testSpace := NewRegularTestSpace(&cfg, quotaLimit)
			Expect(testSpace.Timeout).To(Equal(1 * time.Minute))
		})

		Context("when the config scales the timeout", func() {
			BeforeEach(func() {
				cfg = config.Config{
					NamePrefix:   namePrefix,
					TimeoutScale: 2.0,
				}
			})

			It("scales the timeout for cf commands", func() {
				testSpace := NewRegularTestSpace(&cfg, quotaLimit)
				Expect(testSpace.Timeout).To(Equal(2 * time.Minute))
			})
		})

		It("uses default values for the quota (except for QuotaDefinitionTotalMemoryLimit)", func() {
			testSpace := NewRegularTestSpace(&cfg, quotaLimit)
			Expect(testSpace.QuotaDefinitionInstanceMemoryLimit).To(Equal("-1"))
			Expect(testSpace.QuotaDefinitionRoutesLimit).To(Equal("1000"))
			Expect(testSpace.QuotaDefinitionAppInstanceLimit).To(Equal("-1"))
			Expect(testSpace.QuotaDefinitionServiceInstanceLimit).To(Equal("100"))
			Expect(testSpace.QuotaDefinitionAllowPaidServicesFlag).To(Equal("--allow-paid-service-plans"))
		})

		It("uses the provided QuotaDefinitionTotalMemoryLimit", func() {
			testSpace := NewRegularTestSpace(&cfg, quotaLimit)
			Expect(testSpace.QuotaDefinitionTotalMemoryLimit).To(Equal(quotaLimit))
		})

		It("makes the space ephemeral", func() {
			testSpace := NewRegularTestSpace(&cfg, quotaLimit)
			Expect(testSpace.ShouldRemain()).To(BeFalse())
		})

		Context("when the config specifies that an existing organization should be used", func() {
			BeforeEach(func() {
				cfg = config.Config{
					UseExistingOrganization: true,
					ExistingOrganization:    "existing-org",
				}
			})
			It("uses the provided existing organization name", func() {
				testSpace := NewRegularTestSpace(&cfg, quotaLimit)
				Expect(testSpace.OrganizationName()).To(Equal("existing-org"))
			})
			Context("when the config does not specify the existing organization name", func() {
				BeforeEach(func() {
					cfg = config.Config{
						UseExistingOrganization: true,
					}
				})
				It("panics", func() {
					Expect(func() {
						NewRegularTestSpace(&cfg, quotaLimit)
					}).Should(Panic())
				})
			})
		})

	})

	Describe("NewPersistentAppTestSpace", func() {
		var quotaDefinitionName, organizationName, spaceName string
		BeforeEach(func() {
			quotaDefinitionName = "persistent-quota"
			organizationName = "persistent-org"
			spaceName = "persistent-space"
			cfg = config.Config{
				PersistentAppOrg:       organizationName,
				PersistentAppSpace:     spaceName,
				PersistentAppQuotaName: quotaDefinitionName,
			}
		})

		It("gets the quota definition name from the config", func() {
			testSpace := NewPersistentAppTestSpace(&cfg)
			Expect(testSpace.QuotaDefinitionName).To(Equal(quotaDefinitionName))
		})

		It("gets the org name from the config", func() {
			testSpace := NewPersistentAppTestSpace(&cfg)
			Expect(testSpace.OrganizationName()).To(Equal(organizationName))
		})

		It("gets the space name from the config", func() {
			testSpace := NewPersistentAppTestSpace(&cfg)
			Expect(testSpace.SpaceName()).To(Equal(spaceName))
		})

		It("uses default values for the quota", func() {
			testSpace := NewPersistentAppTestSpace(&cfg)
			Expect(testSpace.QuotaDefinitionTotalMemoryLimit).To(Equal("10G"))
			Expect(testSpace.QuotaDefinitionInstanceMemoryLimit).To(Equal("-1"))
			Expect(testSpace.QuotaDefinitionRoutesLimit).To(Equal("1000"))
			Expect(testSpace.QuotaDefinitionAppInstanceLimit).To(Equal("-1"))
			Expect(testSpace.QuotaDefinitionServiceInstanceLimit).To(Equal("100"))
			Expect(testSpace.QuotaDefinitionAllowPaidServicesFlag).To(Equal("--allow-paid-service-plans"))
		})

		It("makes the space persistent", func() {
			testSpace := NewPersistentAppTestSpace(&cfg)
			Expect(testSpace.ShouldRemain()).To(Equal(true))
		})
	})

	Describe("Create", func() {
		var testSpace *TestSpace
		var fakeStarter *fakes.FakeCmdStarter

		var spaceName, orgName, quotaName, quotaLimit string
		var isPersistent, isExistingOrganization bool
		var timeout time.Duration

		BeforeEach(func() {
			spaceName = "space"
			orgName = "org"
			quotaName = "quota"
			quotaLimit = "10G"
			isPersistent = false
			isExistingOrganization = false
			timeout = 1 * time.Second
			fakeStarter = fakes.NewFakeCmdStarter()
		})

		JustBeforeEach(func() {
			testSpace = NewBaseTestSpace(spaceName, orgName, quotaName, quotaLimit, isPersistent, isExistingOrganization, timeout, fakeStarter)
		})

		It("creates a quota", func() {
			testSpace.Create()
			Expect(len(fakeStarter.CalledWith)).To(BeNumerically(">", 0))
			Expect(fakeStarter.CalledWith[0].Executable).To(Equal("cf"))
			Expect(fakeStarter.CalledWith[0].Args).To(Equal([]string{
				"create-quota", testSpace.QuotaDefinitionName,
				"-m", testSpace.QuotaDefinitionTotalMemoryLimit,
				"-i", testSpace.QuotaDefinitionInstanceMemoryLimit,
				"-r", testSpace.QuotaDefinitionRoutesLimit,
				"-a", testSpace.QuotaDefinitionAppInstanceLimit,
				"-s", testSpace.QuotaDefinitionServiceInstanceLimit,
				testSpace.QuotaDefinitionAllowPaidServicesFlag,
			}))
		})

		It("creates an org", func() {
			testSpace.Create()
			Expect(len(fakeStarter.CalledWith)).To(BeNumerically(">", 1))
			Expect(fakeStarter.CalledWith[1].Executable).To(Equal("cf"))
			Expect(fakeStarter.CalledWith[1].Args).To(Equal([]string{"create-org", testSpace.OrganizationName()}))
		})

		Context("when the config specifies that an existing organization should be used", func() {
			BeforeEach(func() {
				isExistingOrganization = true
			})
			It("does not create the org", func() {
				testSpace.Create()
				for _, calls := range fakeStarter.CalledWith {
					Expect(calls.Args).ToNot(ContainElement("create-org"))
				}
			})
		})

		It("sets quota", func() {
			testSpace.Create()
			Expect(len(fakeStarter.CalledWith)).To(BeNumerically(">", 2))
			Expect(fakeStarter.CalledWith[2].Executable).To(Equal("cf"))
			Expect(fakeStarter.CalledWith[2].Args).To(Equal([]string{"set-quota", testSpace.OrganizationName(), testSpace.QuotaDefinitionName}))
		})

		It("create space", func() {
			testSpace.Create()
			Expect(len(fakeStarter.CalledWith)).To(BeNumerically(">", 3))
			Expect(fakeStarter.CalledWith[3].Executable).To(Equal("cf"))
			Expect(fakeStarter.CalledWith[3].Args).To(Equal([]string{"create-space", "-o", testSpace.OrganizationName(), testSpace.SpaceName()}))
		})

		Describe("failure cases", func() {
			testFailureCase := func(callIndex int) func() {
				return func() {
					BeforeEach(func() {
						fakeStarter.ToReturn[callIndex].ExitCode = 1
					})

					It("returns a ginkgo error", func() {
						failures := InterceptGomegaFailures(func() {
							testSpace.Create()
						})
						Expect(failures).To(HaveLen(1))
						Expect(failures[0]).To(MatchRegexp("to match exit code:\n.*0"))
					})
				}
			}

			Context("when 'cf create-quota' fails", testFailureCase(0))
			Context("when 'cf create-org' fails", testFailureCase(1))
			Context("when 'cf set-quota' fails", testFailureCase(2))
			Context("when 'cf create-space' fails", testFailureCase(3))
		})

		Describe("timing out", func() {
			BeforeEach(func() {
				timeout = 2 * time.Second
			})

			testTimeoutCase := func(callIndex int) func() {
				return func() {
					BeforeEach(func() {
						fakeStarter.ToReturn[callIndex].SleepTime = 5
					})

					It("returns a ginkgo error", func() {
						failures := InterceptGomegaFailures(func() {
							testSpace.Create()
						})

						Expect(failures).To(HaveLen(1))
						Expect(failures[0]).To(MatchRegexp("Timed out after 2.*"))
					})
				}
			}

			Context("when 'cf create-quota' times out", testTimeoutCase(0))
			Context("when 'cf create-org' times out", testTimeoutCase(1))
			Context("when 'cf set-quota' times out", testTimeoutCase(2))
			Context("when 'cf create-space' times out", testTimeoutCase(3))
		})

	})

	Describe("Destroy", func() {
		var testSpace *TestSpace
		var fakeStarter *fakes.FakeCmdStarter
		var spaceName, orgName, quotaName, quotaLimit string
		var isPersistent bool
		var isExistingOrganization bool
		var timeout time.Duration
		BeforeEach(func() {
			fakeStarter = fakes.NewFakeCmdStarter()

			spaceName = "space"
			orgName = "org"
			quotaName = "quota"
			quotaLimit = "10G"
			isPersistent = false
			isExistingOrganization = false
			timeout = 1 * time.Second
		})

		JustBeforeEach(func() {
			testSpace = NewBaseTestSpace(
				spaceName,
				orgName,
				quotaName,
				quotaLimit,
				isPersistent,
				isExistingOrganization,
				timeout,
				fakeStarter,
			)
		})

		It("deletes the org", func() {
			testSpace.Destroy()
			Expect(len(fakeStarter.CalledWith)).To(BeNumerically(">", 0))
			Expect(fakeStarter.CalledWith[0].Executable).To(Equal("cf"))
			Expect(fakeStarter.CalledWith[0].Args).To(Equal([]string{"delete-org", "-f", testSpace.OrganizationName()}))
		})

		It("deletes the quota", func() {
			testSpace.Destroy()
			Expect(len(fakeStarter.CalledWith)).To(BeNumerically(">", 1))
			Expect(fakeStarter.CalledWith[1].Executable).To(Equal("cf"))
			Expect(fakeStarter.CalledWith[1].Args).To(Equal([]string{"delete-quota", "-f", testSpace.QuotaDefinitionName}))
		})

		Context("when the config specifies that an existing organization should be used", func() {
			BeforeEach(func() {
				isExistingOrganization = true
			})
			It("does not delete the org", func() {
				testSpace.Destroy()
				for _, calls := range fakeStarter.CalledWith {
					Expect(calls.Args).ToNot(ContainElement("delete-org"))
				}
			})
			It("deletes the space", func() {
				testSpace.Destroy()
				Expect(len(fakeStarter.CalledWith)).To(BeNumerically(">", 0))
				Expect(fakeStarter.CalledWith[0].Executable).To(Equal("cf"))
				Expect(fakeStarter.CalledWith[0].Args).To(Equal(
					[]string{"delete-space", "-f", "-o", testSpace.OrganizationName(), testSpace.SpaceName()}))
			})
		})

		Describe("failure cases", func() {
			testFailureCase := func(callIndex int) func() {
				return func() {
					BeforeEach(func() {
						fakeStarter.ToReturn[callIndex].ExitCode = 1
					})

					It("returns a ginkgo error", func() {
						failures := InterceptGomegaFailures(func() {
							testSpace.Destroy()
						})
						Expect(failures).To(HaveLen(1))
						Expect(failures[0]).To(MatchRegexp("to match exit code:\n.*0"))
					})
				}
			}

			Context("when 'delete-org' fails", testFailureCase(0))
			Context("when 'delete-quota' fails", testFailureCase(1))
		})

		Describe("timing out", func() {
			BeforeEach(func() {
				timeout = 2 * time.Second
			})

			testTimeoutCase := func(callIndex int) func() {
				return func() {
					BeforeEach(func() {
						fakeStarter.ToReturn[callIndex].SleepTime = 5
					})

					It("returns a ginkgo error", func() {
						failures := InterceptGomegaFailures(func() {
							testSpace.Destroy()
						})

						Expect(failures).To(HaveLen(1))
						Expect(failures[0]).To(MatchRegexp("Timed out after 2.*"))
					})
				}
			}

			Context("when 'cf delete-org' times out", testTimeoutCase(0))
			Context("when 'cf delete-quota' times out", testTimeoutCase(1))
		})
	})

	Describe("ShouldRemain", func() {
		var testSpace *TestSpace
		var isPersistent bool
		JustBeforeEach(func() {
			testSpace = NewBaseTestSpace("", "", "", "", isPersistent, false, 1*time.Second, nil)
		})
		Context("when the space is constructed to be ephemeral", func() {
			BeforeEach(func() {
				isPersistent = false
			})
			It("returns false", func() {
				Expect(testSpace.ShouldRemain()).To(BeFalse())
			})
		})

		Context("when the space is contstructed to be persistent", func() {
			BeforeEach(func() {
				isPersistent = true
			})

			It("returns true", func() {
				Expect(testSpace.ShouldRemain()).To(BeTrue())
			})
		})
	})

	Describe("OrganizationName", func() {
		var testSpace *TestSpace
		BeforeEach(func() {
			testSpace = nil
		})

		It("returns the organization name", func() {
			testSpace = NewBaseTestSpace("", "my-org", "", "", false, false, 1*time.Second, nil)
			Expect(testSpace.OrganizationName()).To(Equal("my-org"))
		})

		Context("when the TestSpace is nil", func() {
			It("returns the empty string", func() {
				Expect(testSpace.OrganizationName()).To(BeEmpty())
			})
		})
	})

	Describe("SpaceName", func() {
		var testSpace *TestSpace
		BeforeEach(func() {
			testSpace = nil
		})

		It("returns the organization name", func() {
			testSpace = NewBaseTestSpace("my-space", "", "", "", false, false, 1*time.Second, nil)
			Expect(testSpace.SpaceName()).To(Equal("my-space"))
		})

		Context("when the TestSpace is nil", func() {
			It("returns the empty string", func() {
				Expect(testSpace.SpaceName()).To(BeEmpty())
			})
		})
	})
})
