# UI coverage: exhaustive source index

154/154 Svelte components indexed; reviewed by feature and used to scope the full UI PRD. Runtime state-specific visual testing is a separate acceptance condition.

| # | Svelte file | Source state flags | Visual scope |
|---:|---|---|---|
| 1 | web/src/app/App.svelte | LC | Out: keep auth/error/shared style |
| 2 | web/src/features/auth/AuthLayout.svelte | - | Out: keep auth/error/shared style |
| 3 | web/src/features/auth/ConsentPage.svelte | LC | Out: keep auth/error/shared style |
| 4 | web/src/features/auth/LoginPage.svelte | XFC | Out: keep auth/error/shared style |
| 5 | web/src/features/auth/RegisterPage.svelte | XFC | Out: keep auth/error/shared style |
| 6 | web/src/features/auth/components/PasswordStrength.svelte | C | Out: keep auth/error/shared style |
| 7 | web/src/features/auth/components/TextField.svelte | XC | Out: keep auth/error/shared style |
| 8 | web/src/features/errors/ErrorPage.svelte | - | Out: keep auth/error/shared style |
| 9 | web/src/features/workspace/changes/ChangedFile.svelte | XC | P2: keep flat chrome, composer, editor or primitive |
| 10 | web/src/features/workspace/changes/DiffView.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 11 | web/src/features/workspace/changes/GitChangeDiff.svelte | LC | P2: keep flat chrome, composer, editor or primitive |
| 12 | web/src/features/workspace/components/AttachmentChips.svelte | LXFMC | P2: keep flat chrome, composer, editor or primitive |
| 13 | web/src/features/workspace/components/AttachmentNotices.svelte | XC | P2: keep flat chrome, composer, editor or primitive |
| 14 | web/src/features/workspace/components/CodeEditor.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 15 | web/src/features/workspace/components/ComposerBox.svelte | LFC | P2: keep flat chrome, composer, editor or primitive |
| 16 | web/src/features/workspace/components/ComposerControls.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 17 | web/src/features/workspace/components/ComposerNotices.svelte | LC | P2: keep flat chrome, composer, editor or primitive |
| 18 | web/src/features/workspace/components/ConfirmModal.svelte | FMC | P2: keep flat chrome, composer, editor or primitive |
| 19 | web/src/features/workspace/components/DocText.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 20 | web/src/features/workspace/components/FormField.svelte | XC | P2: keep flat chrome, composer, editor or primitive |
| 21 | web/src/features/workspace/components/InlineText.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 22 | web/src/features/workspace/components/ProjectTile.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 23 | web/src/features/workspace/components/RoleMentionInput.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 24 | web/src/features/workspace/components/StatusBadge.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 25 | web/src/features/workspace/components/StatusDot.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 26 | web/src/features/workspace/extensions/ExtensionCard.svelte | XC | P2: keep flat chrome, composer, editor or primitive |
| 27 | web/src/features/workspace/extensions/ExtensionsPage.svelte | EXMC | P1: empty/filled consistency; preserve split layout |
| 28 | web/src/features/workspace/extensions/InstallGuide.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 29 | web/src/features/workspace/files/BinaryFile.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 30 | web/src/features/workspace/files/FileBrowser.svelte | ELXFMC | P1: empty/filled consistency; preserve split layout |
| 31 | web/src/features/workspace/files/FileView.svelte | ELXMC | P2: keep flat chrome, composer, editor or primitive |
| 32 | web/src/features/workspace/files/TreeLevel.svelte | XC | P2: keep flat chrome, composer, editor or primitive |
| 33 | web/src/features/workspace/home/Composer.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 34 | web/src/features/workspace/home/HomePage.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 35 | web/src/features/workspace/home/ModelPicker.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 36 | web/src/features/workspace/home/NeedsYou.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 37 | web/src/features/workspace/home/RunningNow.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 38 | web/src/features/workspace/inbox/InboxDetail.svelte | EXC | P2: keep flat chrome, composer, editor or primitive |
| 39 | web/src/features/workspace/inbox/InboxList.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 40 | web/src/features/workspace/inbox/InboxPage.svelte | EC | P1: empty/filled consistency; preserve split layout |
| 41 | web/src/features/workspace/projects/AboutTab.svelte | XFMC | P1: section surfaces; preserve forms/choice cards |
| 42 | web/src/features/workspace/projects/CheckModal.svelte | LXFMC | P1: section surfaces; preserve forms/choice cards |
| 43 | web/src/features/workspace/projects/ChecksTab.svelte | ELXMC | P0: grouped surface or scoped transcript |
| 44 | web/src/features/workspace/projects/IdField.svelte | XC | P1: section surfaces; preserve forms/choice cards |
| 45 | web/src/features/workspace/projects/NameField.svelte | - | P1: section surfaces; preserve forms/choice cards |
| 46 | web/src/features/workspace/projects/NewProjectPage.svelte | LC | P1: section surfaces; preserve forms/choice cards |
| 47 | web/src/features/workspace/projects/ProjectDetailPage.svelte | C | P0: grouped surface or scoped transcript |
| 48 | web/src/features/workspace/projects/ProjectsPage.svelte | EC | P1: section surfaces; preserve forms/choice cards |
| 49 | web/src/features/workspace/projects/ProjectsTable.svelte | C | P1: section surfaces; preserve forms/choice cards |
| 50 | web/src/features/workspace/projects/RepoPicker.svelte | LXC | P1: section surfaces; preserve forms/choice cards |
| 51 | web/src/features/workspace/projects/StartDetails.svelte | XFC | P1: section surfaces; preserve forms/choice cards |
| 52 | web/src/features/workspace/projects/StartGallery.svelte | L | P1: section surfaces; preserve forms/choice cards |
| 53 | web/src/features/workspace/projects/StartSource.svelte | C | P1: section surfaces; preserve forms/choice cards |
| 54 | web/src/features/workspace/projects/StartWhere.svelte | C | P1: section surfaces; preserve forms/choice cards |
| 55 | web/src/features/workspace/projects/TaskRow.svelte | - | P1: section surfaces; preserve forms/choice cards |
| 56 | web/src/features/workspace/projects/TasksTab.svelte | EC | P0: grouped surface or scoped transcript |
| 57 | web/src/features/workspace/projects/egress/AccessPanel.svelte | LXFC | P1: section surfaces; preserve forms/choice cards |
| 58 | web/src/features/workspace/projects/egress/GrantRow.svelte | C | P1: section surfaces; preserve forms/choice cards |
| 59 | web/src/features/workspace/projects/env/DatabaseSection.svelte | LXMC | P1: section surfaces; preserve forms/choice cards |
| 60 | web/src/features/workspace/projects/env/GatewaySection.svelte | LXC | P1: section surfaces; preserve forms/choice cards |
| 61 | web/src/features/workspace/projects/env/VariablesSection.svelte | ELXFMC | P1: section surfaces; preserve forms/choice cards |
| 62 | web/src/features/workspace/settings/AccountTab.svelte | XMC | P1: section surfaces; preserve forms/choice cards |
| 63 | web/src/features/workspace/settings/ActivityTab.svelte | ELXMC | P1: section surfaces; preserve forms/choice cards |
| 64 | web/src/features/workspace/settings/AddProviderModal.svelte | XFMC | P1: section surfaces; preserve forms/choice cards |
| 65 | web/src/features/workspace/settings/ConnectedAppsTab.svelte | ELXMC | P1: section surfaces; preserve forms/choice cards |
| 66 | web/src/features/workspace/settings/ConnectorAddress.svelte | - | P1: section surfaces; preserve forms/choice cards |
| 67 | web/src/features/workspace/settings/ContextWindows.svelte | C | P1: section surfaces; preserve forms/choice cards |
| 68 | web/src/features/workspace/settings/DatabaseCertificates.svelte | C | P1: section surfaces; preserve forms/choice cards |
| 69 | web/src/features/workspace/settings/DatabaseTab.svelte | LXFMC | P1: section surfaces; preserve forms/choice cards |
| 70 | web/src/features/workspace/settings/GitHubIntegration.svelte | LMC | P1: section surfaces; preserve forms/choice cards |
| 71 | web/src/features/workspace/settings/IntegrationsTab.svelte | - | P1: section surfaces; preserve forms/choice cards |
| 72 | web/src/features/workspace/settings/ModelsTab.svelte | ELXMC | P1: section surfaces; preserve forms/choice cards |
| 73 | web/src/features/workspace/settings/NotificationsTab.svelte | C | P1: section surfaces; preserve forms/choice cards |
| 74 | web/src/features/workspace/settings/PersonaTab.svelte | LXC | P1: section surfaces; preserve forms/choice cards |
| 75 | web/src/features/workspace/settings/ProviderRow.svelte | C | P1: section surfaces; preserve forms/choice cards |
| 76 | web/src/features/workspace/settings/RulesList.svelte | F | P1: section surfaces; preserve forms/choice cards |
| 77 | web/src/features/workspace/settings/SettingsFrame.svelte | - | P1: section surfaces; preserve forms/choice cards |
| 78 | web/src/features/workspace/settings/SettingsNav.svelte | - | P1: section surfaces; preserve forms/choice cards |
| 79 | web/src/features/workspace/settings/SettingsPage.svelte | C | P1: section surfaces; preserve forms/choice cards |
| 80 | web/src/features/workspace/settings/SkipApprovals.svelte | XC | P1: section surfaces; preserve forms/choice cards |
| 81 | web/src/features/workspace/settings/SpaceTools.svelte | XC | P1: section surfaces; preserve forms/choice cards |
| 82 | web/src/features/workspace/settings/SpacesTab.svelte | ELXC | P1: section surfaces; preserve forms/choice cards |
| 83 | web/src/features/workspace/settings/SubscriptionDialog.svelte | LXFMC | P1: section surfaces; preserve forms/choice cards |
| 84 | web/src/features/workspace/settings/TeamRole.svelte | XC | P1: section surfaces; preserve forms/choice cards |
| 85 | web/src/features/workspace/settings/TeamTab.svelte | LXC | P1: section surfaces; preserve forms/choice cards |
| 86 | web/src/features/workspace/settings/TierRow.svelte | C | P1: section surfaces; preserve forms/choice cards |
| 87 | web/src/features/workspace/settings/UsageTab.svelte | ELXC | P1: section surfaces; preserve forms/choice cards |
| 88 | web/src/features/workspace/settings/parts/ChangePasswordForm.svelte | XFC | P1: section surfaces; preserve forms/choice cards |
| 89 | web/src/features/workspace/settings/parts/DeleteAccountModal.svelte | XFMC | P1: section surfaces; preserve forms/choice cards |
| 90 | web/src/features/workspace/settings/parts/IntegrationLogo.svelte | - | P1: section surfaces; preserve forms/choice cards |
| 91 | web/src/features/workspace/settings/parts/IntegrationRow.svelte | X | P1: section surfaces; preserve forms/choice cards |
| 92 | web/src/features/workspace/settings/parts/SaveBar.svelte | XC | P1: section surfaces; preserve forms/choice cards |
| 93 | web/src/features/workspace/settings/parts/SettingRow.svelte | C | P1: section surfaces; preserve forms/choice cards |
| 94 | web/src/features/workspace/settings/parts/SettingsSection.svelte | C | P1: section surfaces; preserve forms/choice cards |
| 95 | web/src/features/workspace/shell/AppShell.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 96 | web/src/features/workspace/shell/CommandPalette.svelte | FM | P2: keep flat chrome, composer, editor or primitive |
| 97 | web/src/features/workspace/shell/MobileBar.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 98 | web/src/features/workspace/shell/NavItem.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 99 | web/src/features/workspace/shell/NavItemBody.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 100 | web/src/features/workspace/shell/Sidebar.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 101 | web/src/features/workspace/shell/SidebarPinned.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 102 | web/src/features/workspace/shell/SidebarProjects.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 103 | web/src/features/workspace/shell/SidebarRecent.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 104 | web/src/features/workspace/shell/SidebarSites.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 105 | web/src/features/workspace/shell/SidebarTaskRow.svelte | XC | P2: keep flat chrome, composer, editor or primitive |
| 106 | web/src/features/workspace/shell/Topbar.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 107 | web/src/features/workspace/shell/UserMenu.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 108 | web/src/features/workspace/sites/SiteRow.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 109 | web/src/features/workspace/sites/SitesPage.svelte | EXMC | P1: empty/filled consistency; preserve split layout |
| 110 | web/src/features/workspace/task/ContextMeter.svelte | XC | P2: keep flat chrome, composer, editor or primitive |
| 111 | web/src/features/workspace/task/ReviewBar.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 112 | web/src/features/workspace/task/TaskHeader.svelte | C | P2: keep flat chrome, composer, editor or primitive |
| 113 | web/src/features/workspace/task/TaskPage.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 114 | web/src/features/workspace/task/TaskScreen.svelte | LXC | P0: grouped surface or scoped transcript |
| 115 | web/src/features/workspace/task/panel/ChangesTab.svelte | ELC | P1: empty/filled consistency; preserve split layout |
| 116 | web/src/features/workspace/task/panel/ExternalTeam.svelte | EC | P1: empty/filled consistency; preserve split layout |
| 117 | web/src/features/workspace/task/panel/FilesTab.svelte | - | P2: keep flat chrome, composer, editor or primitive |
| 118 | web/src/features/workspace/task/panel/PlanTab.svelte | XC | P0: grouped surface or scoped transcript |
| 119 | web/src/features/workspace/task/panel/ProjectDocuments.svelte | LXC | P0: grouped surface or scoped transcript |
| 120 | web/src/features/workspace/task/panel/TeamTab.svelte | EC | P1: empty/filled consistency; preserve split layout |
| 121 | web/src/features/workspace/task/panel/WorkPanel.svelte | C | P0: grouped surface or scoped transcript |
| 122 | web/src/features/workspace/task/thread/ActionGroup.svelte | C | P2: keep semantic event cards |
| 123 | web/src/features/workspace/task/thread/ApprovalCard.svelte | C | P2: keep semantic event cards |
| 124 | web/src/features/workspace/task/thread/ChatBubble.svelte | C | P2: keep semantic event cards |
| 125 | web/src/features/workspace/task/thread/EventItem.svelte | C | P0: grouped surface or scoped transcript |
| 126 | web/src/features/workspace/task/thread/EvidenceBadge.svelte | C | P2: keep semantic event cards |
| 127 | web/src/features/workspace/task/thread/ExtensionCard.svelte | - | P2: keep semantic event cards |
| 128 | web/src/features/workspace/task/thread/HandoffCard.svelte | XC | P2: keep semantic event cards |
| 129 | web/src/features/workspace/task/thread/PlanCard.svelte | - | P2: keep semantic event cards |
| 130 | web/src/features/workspace/task/thread/PlanReviewCard.svelte | LXC | P2: keep semantic event cards |
| 131 | web/src/features/workspace/task/thread/PrdCard.svelte | C | P2: keep semantic event cards |
| 132 | web/src/features/workspace/task/thread/PreviewChoice.svelte | LC | P2: keep semantic event cards |
| 133 | web/src/features/workspace/task/thread/QuestionCard.svelte | FC | P2: keep semantic event cards |
| 134 | web/src/features/workspace/task/thread/QuestionField.svelte | C | P2: keep semantic event cards |
| 135 | web/src/features/workspace/task/thread/ResultCard.svelte | XC | P2: keep semantic event cards |
| 136 | web/src/features/workspace/task/thread/RoleCard.svelte | - | P2: keep semantic event cards |
| 137 | web/src/features/workspace/task/thread/SentFiles.svelte | LC | P2: keep semantic event cards |
| 138 | web/src/features/workspace/task/thread/ShotCard.svelte | LC | P2: keep semantic event cards |
| 139 | web/src/features/workspace/task/thread/SiteCard.svelte | - | P2: keep semantic event cards |
| 140 | web/src/features/workspace/task/thread/TaskThread.svelte | LC | P0: grouped surface or scoped transcript |
| 141 | web/src/features/workspace/tasks/ArchivePage.svelte | EXC | P0: grouped surface or scoped transcript |
| 142 | web/src/features/workspace/tasks/TaskSelection.svelte | MC | P0: grouped surface or scoped transcript |
| 143 | web/src/features/workspace/tasks/TasksPage.svelte | EC | P0: grouped surface or scoped transcript |
| 144 | web/src/shared/forms/PasswordField.svelte | XC | Out: keep auth/error/shared style |
| 145 | web/src/shared/ui/BrandLink.svelte | - | Out: keep auth/error/shared style |
| 146 | web/src/shared/ui/EmptyState.svelte | C | Out: keep auth/error/shared style |
| 147 | web/src/shared/ui/Icon.svelte | C | Out: keep auth/error/shared style |
| 148 | web/src/shared/ui/Illustration.svelte | L | Out: keep auth/error/shared style |
| 149 | web/src/shared/ui/LazySection.svelte | C | Out: keep auth/error/shared style |
| 150 | web/src/shared/ui/LoadingScreen.svelte | L | Out: keep auth/error/shared style |
| 151 | web/src/shared/ui/LogoMark.svelte | - | Out: keep auth/error/shared style |
| 152 | web/src/shared/ui/Marker.svelte | - | Out: keep auth/error/shared style |
| 153 | web/src/shared/ui/SegmentedControl.svelte | - | Out: keep auth/error/shared style |
| 154 | web/src/shared/ui/SkeletonList.svelte | C | Out: keep auth/error/shared style |
