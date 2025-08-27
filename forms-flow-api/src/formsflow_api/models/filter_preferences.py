"""This manages filter preference Database Models."""

from __future__ import annotations

from typing import List

from formsflow_api_utils.utils.enums import FilterStatus
from sqlalchemy import Index, UniqueConstraint, and_, or_, tuple_
from sqlalchemy.orm import relationship

from .audit_mixin import AuditDateTimeMixin
from .base_model import BaseModel
from .db import db
from .filter import Filter


class FilterPreferences(db.Model, BaseModel, AuditDateTimeMixin):
    """Stores user-specific preferences for filters, including sort order and visibility."""

    __tablename__ = "filter_preferences"
    id = db.Column(
        db.Integer,
        primary_key=True,
        comment="Primary key for the User Preference entry.",
    )
    tenant = db.Column(
        db.String,
        nullable=True,
        comment="Tenant identifier (optional, for multi-tenant support).",
    )
    user_id = db.Column(
        db.String, index=True, nullable=False, comment="Unique identifier for the user."
    )
    filter_id = db.Column(
        db.Integer,
        db.ForeignKey("filter.id", ondelete="CASCADE"),
        comment="Reference to the filter ID.",
    )
    sort_order = db.Column(
        db.Integer, comment="Sort order preference for the applied filter."
    )
    hide = db.Column(
        db.Boolean,
        default=False,
        nullable=False,
        comment="Flag to indicate if the filter is hidden.",
    )

    filter = relationship("Filter", lazy="joined", backref="filter_preferences")

    __table_args__ = (
        UniqueConstraint("user_id", "filter_id", name="_user_filter_uc"),
        Index("idx_user_id_and_tenant", "tenant", "user_id"),
    )

    @classmethod
    def bulk_upsert_preferences(cls, preferences_list: List[dict], tenant_key: str):
        """Upsert in filter preferences."""
        if not preferences_list:
            return []

        # exctract userid and filter id for search
        keys = [(p["user_id"], p["filter_id"]) for p in preferences_list]
        # fetch existing data
        query = cls.query.filter(tuple_(cls.user_id, cls.filter_id).in_(keys))
        if tenant_key:
            query.filter(cls.tenant == tenant_key)
        # feth all existing records
        existing_records = query.all()
        # create a existing data lookup dict
        existing_lookup = {
            (r.user_id, r.filter_id, r.tenant): r for r in existing_records
        }
        for pref in preferences_list:
            key = (pref["user_id"], pref["filter_id"], pref.get("tenant"))
            if key in existing_lookup:
                # Update existing
                record = existing_lookup[key]
                record.sort_order = pref.get("sort_order")
                record.hide = pref.get("hide", False)
            else:
                # Create new
                new_record = cls(
                    user_id=pref["user_id"],
                    filter_id=pref["filter_id"],
                    sort_order=pref.get("sort_order"),
                    tenant=pref.get("tenant"),
                    hide=pref.get("hide", False),
                )
                db.session.add(new_record)
        return db.session.commit()

    @classmethod
    def get_filters_by_user_id(  # pylint: disable-msg=too-many-arguments, too-many-locals, too-many-positional-arguments
        cls,
        user_id: str,
        tenant: str,
        filter_type: str,
        roles: List[str],
        parent_filter_id: int = None,
    ) -> List[FilterPreferences]:
        """Find filter preference with specific user id - SIMPLIFIED OPTIMIZATION.
        
        Focus on the core bottleneck: eliminate complex array operations and use simpler queries.
        """
        import time
        import logging
        
        perf_logger = logging.getLogger('performance.filter_preferences_detailed')
        start_time = time.time()
        
        perf_logger.info(f"[PERF] get_filters_by_user_id SIMPLIFIED started for user: {user_id}")
        
        # STEP 1: Get basic filter preferences for the user (this should be fast)
        basic_start = time.time()
        base_query = cls.query.filter(cls.user_id == user_id)
        
        if tenant:
            base_query = base_query.filter(cls.tenant == tenant)
        
        basic_duration = (time.time() - basic_start) * 1000
        perf_logger.info(f"[PERF] Basic preference filter: {basic_duration:.2f}ms")
        
        # STEP 2: Apply filter table constraints with explicit JOIN
        join_start = time.time()
        query = base_query.join(Filter, cls.filter_id == Filter.id)
        
        # Apply simple, indexed filters
        query = query.filter(
            Filter.status == FilterStatus.ACTIVE.value,
            Filter.filter_type == filter_type,
        )
        
        if parent_filter_id:
            query = query.filter(Filter.parent_filter_id == parent_filter_id)
        
        join_duration = (time.time() - join_start) * 1000
        perf_logger.info(f"[PERF] JOIN and basic filters: {join_duration:.2f}ms")
        
        # STEP 3: Authorization check - SINGLE OPTIMIZED QUERY
        auth_start = time.time()
        
        # CRITICAL FIX: Combine into ONE query to eliminate network round trips
        # Use OR to get both user-created and public filters in a single database hit
        combined_query = query.filter(
            or_(
                # User-created filters
                Filter.created_by == user_id,
                # Public filters (no role/user restrictions)
                and_(
                    or_(Filter.roles.is_(None), Filter.roles == []),
                    or_(Filter.users.is_(None), Filter.users == [])
                )
            )
        )
        
        # Single database query instead of two separate ones
        result = combined_query.order_by(cls.sort_order.asc()).all()
        
        auth_duration = (time.time() - auth_start) * 1000
        perf_logger.info(f"[PERF] Authorization (simplified): {auth_duration:.2f}ms")
        
        total_duration = (time.time() - start_time) * 1000
        perf_logger.info(f"[PERF] get_filters_by_user_id SIMPLIFIED TOTAL: {total_duration:.2f}ms, returned {len(result)} results")
        perf_logger.info(f"[PERF] BREAKDOWN - Basic: {basic_duration:.1f}ms, "
                        f"JOIN: {join_duration:.1f}ms, Auth: {auth_duration:.1f}ms")
        
        return result
